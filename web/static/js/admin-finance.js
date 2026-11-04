document.addEventListener("DOMContentLoaded", () => {
  const confirmationForms = [...document.querySelectorAll("form[data-confirm], form[action$='/delete']")];
  confirmationForms.forEach((form) => {
    form.addEventListener("submit", async (event) => {
      if (form.dataset.confirmed === "true") return;
      event.preventDefault();
      event.stopImmediatePropagation();

      const legacyConfirmation = form.getAttribute("onsubmit")?.match(/confirm\(['\"](.+?)['\"]\)/)?.[1];
      const message = form.dataset.confirm || legacyConfirmation || "Data yang dipilih akan dihapus secara permanen.";
      if (typeof window.Swal === "undefined") {
        if (window.confirm(message)) form.submit();
        return;
      }

      const result = await window.Swal.fire({
        title: form.dataset.confirmTitle || "Hapus data?",
        text: message,
        icon: "warning",
        showCancelButton: true,
        confirmButtonText: form.dataset.confirmButton || "Ya, hapus",
        cancelButtonText: "Batal",
        confirmButtonColor: "#dc2626",
        cancelButtonColor: "#64748b",
        reverseButtons: true,
        focusCancel: true,
        customClass: { popup: "siato-swal-popup" }
      });

      if (result.isConfirmed) {
        form.dataset.confirmed = "true";
        form.submit();
      }
    }, true);
  });

  const dropdowns = [...document.querySelectorAll("[data-admin-dropdown]")];
  document.addEventListener("click", (event) => {
    dropdowns.forEach((dropdown) => {
      if (!dropdown.contains(event.target)) dropdown.removeAttribute("open");
    });
  });

  const markRead = document.querySelector("[data-mark-notifications-read]");
  markRead?.addEventListener("click", () => {
    document.querySelectorAll("[data-unread-notification]").forEach((item) => item.classList.remove("bg-violet-50/60"));
    document.querySelectorAll("[data-unread-dot]").forEach((dot) => dot.remove());
    const count = document.querySelector("[data-notification-count]");
    if (count) count.remove();
    const label = document.querySelector("[data-unread-label]");
    if (label) label.textContent = "Semua sudah dibaca";
  });

  const groups = [...document.querySelectorAll("[data-sidebar-group]")];
  groups.forEach((group, index) => {
    const key = `siato-sidebar-${index}`;
    if (localStorage.getItem(key) === "open") group.open = true;
    group.addEventListener("toggle", () => localStorage.setItem(key, group.open ? "open" : "closed"));
  });

  const currentPath = window.location.pathname;
  const sidebarLinks = [...document.querySelectorAll("aside nav a[href]")];
  sidebarLinks.forEach((link) => {
    const href = link.getAttribute("href");
    if (!href || href === "#") return;
    const active = href === "/dashboard" ? currentPath === href : currentPath === href || currentPath.startsWith(`${href}/`);
    if (href === "/dashboard" && !active) {
      link.classList.remove("bg-[#8e1a8e]", "font-bold", "text-white", "shadow-[0_5px_10px_rgba(147,32,147,.22)]");
      link.classList.add("text-[#173254]", "hover:bg-[#fbf3fb]", "hover:text-[#800080]");
      link.querySelector("span.absolute")?.classList.add("hidden");
    }
    if (active && href !== "/dashboard") {
      link.classList.add("bg-violet-50", "font-bold", "text-violet-800");
      link.setAttribute("aria-current", "page");
      const group = link.closest("details");
      if (group) {
        group.open = true;
        const summary = group.querySelector(":scope > summary");
        group.dataset.active = "true";
        summary?.setAttribute("aria-current", "true");
      }
    }
  });

  if (currentPath === "/users") {
    document.querySelectorAll("main table tbody tr").forEach((row, index) => {
      const numberCell = row.querySelector("td");
      if (numberCell) numberCell.textContent = String(index + 1);
    });
  }

  const permissionCheckboxes = [...document.querySelectorAll("[data-permission-checkbox]")];
  const selectAllPermissions = document.querySelector("[data-select-all-permissions]");
  const selectedPermissionCount = document.querySelector("[data-selected-permission-count]");
  const refreshPermissionState = () => {
    const selected = permissionCheckboxes.filter((checkbox) => checkbox.checked).length;
    if (selectedPermissionCount) selectedPermissionCount.textContent = String(selected);
    if (selectAllPermissions) {
      const allSelected = permissionCheckboxes.length > 0 && selected === permissionCheckboxes.length;
      selectAllPermissions.textContent = allSelected ? "Batalkan Semua" : "Pilih Semua";
    }
  };
  permissionCheckboxes.forEach((checkbox) => checkbox.addEventListener("change", refreshPermissionState));
  selectAllPermissions?.addEventListener("click", () => {
    const shouldSelect = permissionCheckboxes.some((checkbox) => !checkbox.checked);
    permissionCheckboxes.forEach((checkbox) => { checkbox.checked = shouldSelect; });
    refreshPermissionState();
  });
  refreshPermissionState();

  document.addEventListener("keydown", (event) => {
    if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === "k") {
      event.preventDefault();
      document.querySelector('input[type="search"]')?.focus();
    }
  });
});
