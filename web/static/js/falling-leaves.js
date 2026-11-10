(() => {
  "use strict";
  const canvas = document.getElementById("falling-leaves");
  if (!canvas || matchMedia("(prefers-reduced-motion: reduce)").matches) return;
  const ctx = canvas.getContext("2d", { alpha: true });
  if (!ctx) return;

  const colors = [[164,42,26],[198,69,24],[224,112,24],[236,155,35],[139,83,32],[116,133,45]];
  const leaves = [];
  let width = 0, height = 0, frame = 0, previous = performance.now();
  const random = (min, max) => min + Math.random() * (max - min);

  function createLeaf(initial = false) {
    const depth = random(.55, 1.35);
    return {
      x: random(-80, width + 80), y: initial ? random(-height, height) : random(-160, -35),
      size: random(11, 21) * depth, speed: random(30, 64) * depth,
      drift: random(-16, 16), phase: random(0, Math.PI * 2), sway: random(18, 54),
      swaySpeed: random(.55, 1.25), rotation: random(0, Math.PI * 2), spin: random(-1.3, 1.3),
      flip: random(0, Math.PI * 2), flipSpeed: random(1.1, 2.5), tilt: random(-.65, .65),
      opacity: random(.48, .88), maple: Math.random() < .55,
      color: colors[Math.floor(Math.random() * colors.length)]
    };
  }

  function resize() {
    const dpr = Math.min(devicePixelRatio || 1, 2);
    width = innerWidth; height = innerHeight;
    canvas.width = Math.round(width * dpr); canvas.height = Math.round(height * dpr);
    canvas.style.width = `${width}px`; canvas.style.height = `${height}px`;
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    const target = Math.max(12, Math.min(width < 640 ? 18 : 34, Math.round(width * height / 52000)));
    while (leaves.length < target) leaves.push(createLeaf(true));
    leaves.length = Math.min(leaves.length, target);
  }

  function maple(size) {
    const points = [[0,-1],[.15,-.48],[.48,-.72],[.39,-.28],[.86,-.22],[.48,.08],[.61,.49],[.12,.28],[0,.92],[-.12,.28],[-.61,.49],[-.48,.08],[-.86,-.22],[-.39,-.28],[-.48,-.72],[-.15,-.48]];
    ctx.beginPath(); ctx.moveTo(points[0][0] * size, points[0][1] * size);
    for (let i = 1; i < points.length; i++) ctx.lineTo(points[i][0] * size, points[i][1] * size);
    ctx.closePath();
  }

  function ordinary(size) {
    ctx.beginPath(); ctx.moveTo(0, -size);
    ctx.bezierCurveTo(size * .72, -size * .55, size * .7, size * .48, 0, size);
    ctx.bezierCurveTo(-size * .7, size * .48, -size * .72, -size * .55, 0, -size); ctx.closePath();
  }

  function draw(leaf, time) {
    const flip = Math.cos(leaf.flip + time * leaf.flipSpeed);
    const [r, g, b] = leaf.color;
    ctx.save(); ctx.translate(leaf.x, leaf.y);
    ctx.rotate(leaf.rotation + Math.sin(time * .8 + leaf.phase) * .22);
    ctx.transform(1, Math.sin(leaf.tilt) * .22, 0, flip, 0, 0);
    ctx.shadowColor = "rgba(55,32,18,.24)"; ctx.shadowBlur = 5 + leaf.size * .18;
    ctx.shadowOffsetX = 2; ctx.shadowOffsetY = 4;
    leaf.maple ? maple(leaf.size) : ordinary(leaf.size);
    const gradient = ctx.createLinearGradient(-leaf.size, -leaf.size, leaf.size, leaf.size);
    gradient.addColorStop(0, `rgba(${Math.min(255,r+35)},${Math.min(255,g+25)},${Math.min(255,b+14)},${leaf.opacity})`);
    gradient.addColorStop(.5, `rgba(${r},${g},${b},${leaf.opacity})`);
    gradient.addColorStop(1, `rgba(${r*.58},${g*.58},${b*.58},${leaf.opacity * (.7 + Math.abs(flip) * .3)})`);
    ctx.fillStyle = gradient; ctx.fill(); ctx.shadowColor = "transparent";
    ctx.strokeStyle = `rgba(89,53,24,${leaf.opacity * .55})`; ctx.lineWidth = Math.max(.6, leaf.size * .045);
    ctx.beginPath(); ctx.moveTo(0, -leaf.size * .7); ctx.lineTo(0, leaf.size * .92); ctx.stroke(); ctx.restore();
  }

  function animate(now) {
    const delta = Math.min((now - previous) / 1000, .04), time = now / 1000; previous = now;
    ctx.clearRect(0, 0, width, height);
    leaves.forEach((leaf, index) => {
      leaf.x += (leaf.drift + Math.sin(time * leaf.swaySpeed + leaf.phase) * leaf.sway * .55) * delta;
      leaf.y += leaf.speed * delta * (1 + Math.sin(time * 1.4 + leaf.phase) * .08);
      leaf.rotation += leaf.spin * delta; draw(leaf, time);
      if (leaf.y > height + leaf.size * 2 || leaf.x < -180 || leaf.x > width + 180) leaves[index] = createLeaf();
    });
    frame = requestAnimationFrame(animate);
  }

  resize(); addEventListener("resize", resize, { passive: true });
  document.addEventListener("visibilitychange", () => {
    cancelAnimationFrame(frame);
    if (!document.hidden) { previous = performance.now(); frame = requestAnimationFrame(animate); }
  });
  frame = requestAnimationFrame(animate);
})();
