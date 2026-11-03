-- phpMyAdmin SQL Dump
-- version 5.2.1
-- https://www.phpmyadmin.net/
--
-- Host: 127.0.0.1
-- Generation Time: Nov 03, 2026 at 03:18 PM
-- Server version: 10.4.28-MariaDB
-- PHP Version: 8.2.4

SET SQL_MODE = "NO_AUTO_VALUE_ON_ZERO";
START TRANSACTION;
SET time_zone = "+00:00";


/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8mb4 */;

--
-- Database: `siato_spc_go`
--

-- --------------------------------------------------------

--
-- Table structure for table `academic_periods`
--

CREATE TABLE `academic_periods` (
  `id` bigint(20) NOT NULL,
  `code` varchar(30) NOT NULL,
  `academic_year` varchar(9) NOT NULL,
  `semester` varchar(20) NOT NULL,
  `start_date` date NOT NULL,
  `end_date` date NOT NULL,
  `uts_date` date DEFAULT NULL,
  `uas_date` date DEFAULT NULL,
  `is_active` tinyint(1) NOT NULL DEFAULT 0,
  `created_at` datetime(6) NOT NULL,
  `updated_at` datetime(6) NOT NULL
) ;

--
-- Dumping data for table `academic_periods`
--

INSERT INTO `academic_periods` (`id`, `code`, `academic_year`, `semester`, `start_date`, `end_date`, `uts_date`, `uas_date`, `is_active`, `created_at`, `updated_at`) VALUES
(1, '2026-GANJIL', '2026/2027', 'GANJIL', '2026-09-01', '2027-02-28', '2026-10-26', '2027-02-01', 1, '2026-09-05 10:19:06.713457', '2026-09-05 10:19:06.713457');

-- --------------------------------------------------------

--
-- Table structure for table `audit_logs`
--

CREATE TABLE `audit_logs` (
  `id` bigint(20) NOT NULL,
  `entity_type` varchar(100) NOT NULL,
  `entity_id` bigint(20) NOT NULL,
  `action` varchar(50) NOT NULL,
  `old_values_json` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`old_values_json`)),
  `new_values_json` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`new_values_json`)),
  `reason` varchar(500) DEFAULT NULL,
  `performed_by_id` varchar(255) DEFAULT NULL,
  `performed_at` datetime(6) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- Table structure for table `billing_policies`
--

CREATE TABLE `billing_policies` (
  `id` bigint(20) NOT NULL,
  `code` varchar(50) NOT NULL,
  `name` varchar(150) NOT NULL,
  `charge_type_id` bigint(20) NOT NULL,
  `academic_period_id` bigint(20) NOT NULL,
  `study_program_id` bigint(20) DEFAULT NULL,
  `student_category_id` bigint(20) DEFAULT NULL,
  `cohort_year` int(11) DEFAULT NULL,
  `cohort_year_class` varchar(30) DEFAULT NULL,
  `payment_method` varchar(20) NOT NULL,
  `total_amount` decimal(18,2) NOT NULL,
  `installment_count` int(11) NOT NULL DEFAULT 1,
  `notes` varchar(500) DEFAULT NULL,
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `created_at` datetime(6) NOT NULL,
  `updated_at` datetime(6) NOT NULL
) ;

--
-- Dumping data for table `billing_policies`
--

INSERT INTO `billing_policies` (`id`, `code`, `name`, `charge_type_id`, `academic_period_id`, `study_program_id`, `student_category_id`, `cohort_year`, `cohort_year_class`, `payment_method`, `total_amount`, `installment_count`, `notes`, `is_active`, `created_at`, `updated_at`) VALUES
(1, 'SPP-TI-REG-2026-6X', 'SPP TI Reguler - Bulanan 6 Kali', 1, 1, 1, 1, 2026, '2026.1.A', 'MONTHLY', 3000000.00, 6, 'Contoh pembayaran tetap Rp500.000 setiap bulan.', 1, '2026-09-05 10:19:06.713457', '2026-09-05 10:19:06.713457'),
(2, 'SPP-SI-REG-2026-UTS-UAS', 'SPP SI Reguler - Sebelum UTS dan UAS', 1, 1, 2, 1, 2026, NULL, 'MILESTONE', 4000000.00, 2, 'Contoh pembayaran 50% sebelum UTS dan 50% sebelum UAS.', 1, '2026-09-05 10:19:06.713457', '2026-09-05 10:19:06.713457'),
(3, 'SPP-TI-KARYAWAN-2025-3X', 'SPP Anak Karyawan - Cicilan 3 Kali', 1, 1, 1, 3, 2025, '2025.1.A', 'INSTALLMENT', 1500000.00, 3, 'Contoh tarif khusus anak karyawan dengan tiga kali cicilan.', 1, '2026-09-05 10:19:06.713457', '2026-09-05 10:19:06.713457'),
(4, 'SPP-SI-MARKETING-2025-CUSTOM', 'SPP Khusus Marketing - Jadwal Custom', 1, 1, 2, 4, 2025, '2025.1.B', 'CUSTOM', 2400000.00, 4, 'Contoh nominal dan tanggal cicilan tidak sama sesuai persetujuan marketing.', 1, '2026-09-05 10:19:06.713457', '2026-09-05 10:19:06.713457'),
(5, 'SPP-SI-BEASISWA-2026', 'SPP SI Beasiswa - Ditanggung 100%', 1, 1, 2, 2, 2026, '2026.1.B', 'MILESTONE', 4000000.00, 2, 'Tarif dasar tetap dicatat, sedangkan nilai yang dibayar mahasiswa ditetapkan Rp0 karena beasiswa 100%.', 1, '2026-09-05 10:19:06.713457', '2026-09-05 10:19:06.713457'),
(6, 'SPP-TI-REG-2024-2X', 'SPP TI Reguler Angkatan 2024 - 2 Kali', 1, 1, 1, 1, 2024, '2024.1.A', 'INSTALLMENT', 3000000.00, 2, 'Contoh kebijakan reguler untuk mahasiswa angkatan 2024.', 1, '2026-09-05 10:19:06.713457', '2026-09-05 10:19:06.713457');

-- --------------------------------------------------------

--
-- Table structure for table `billing_policy_schedules`
--

CREATE TABLE `billing_policy_schedules` (
  `id` bigint(20) NOT NULL,
  `billing_policy_id` bigint(20) NOT NULL,
  `sequence_no` int(11) NOT NULL,
  `description` varchar(150) NOT NULL,
  `due_date` date NOT NULL,
  `amount` decimal(18,2) NOT NULL,
  `percentage` decimal(5,2) DEFAULT NULL,
  `created_at` datetime(6) NOT NULL,
  `updated_at` datetime(6) NOT NULL
) ;

--
-- Dumping data for table `billing_policy_schedules`
--

INSERT INTO `billing_policy_schedules` (`id`, `billing_policy_id`, `sequence_no`, `description`, `due_date`, `amount`, `percentage`, `created_at`, `updated_at`) VALUES
(1, 1, 1, 'Cicilan September', '2026-09-10', 500000.00, 16.67, '2026-09-05 10:19:06.713457', '2026-09-05 10:19:06.713457'),
(2, 1, 2, 'Cicilan Oktober', '2026-10-10', 500000.00, 16.67, '2026-09-05 10:19:06.713457', '2026-09-05 10:19:06.713457'),
(3, 1, 3, 'Cicilan November', '2026-11-10', 500000.00, 16.67, '2026-09-05 10:19:06.713457', '2026-09-05 10:19:06.713457'),
(4, 1, 4, 'Cicilan Desember', '2026-12-10', 500000.00, 16.67, '2026-09-05 10:19:06.713457', '2026-09-05 10:19:06.713457'),
(5, 1, 5, 'Cicilan Januari', '2027-01-10', 500000.00, 16.66, '2026-09-05 10:19:06.713457', '2026-09-05 10:19:06.713457'),
(6, 1, 6, 'Cicilan Februari', '2027-02-10', 500000.00, 16.66, '2026-09-05 10:19:06.713457', '2026-09-05 10:19:06.713457'),
(7, 2, 1, 'Termin sebelum UTS', '2026-10-20', 2000000.00, 50.00, '2026-09-05 10:19:06.713457', '2026-09-05 10:19:06.713457'),
(8, 2, 2, 'Termin sebelum UAS', '2027-01-25', 2000000.00, 50.00, '2026-09-05 10:19:06.713457', '2026-09-05 10:19:06.713457'),
(9, 3, 1, 'Cicilan pertama', '2026-09-15', 500000.00, 33.34, '2026-09-05 10:19:06.713457', '2026-09-05 10:19:06.713457'),
(10, 3, 2, 'Cicilan kedua', '2026-11-15', 500000.00, 33.33, '2026-09-05 10:19:06.713457', '2026-09-05 10:19:06.713457'),
(11, 3, 3, 'Cicilan ketiga', '2027-01-15', 500000.00, 33.33, '2026-09-05 10:19:06.713457', '2026-09-05 10:19:06.713457'),
(12, 4, 1, 'Pembayaran awal', '2026-09-20', 400000.00, 16.67, '2026-09-05 10:19:06.713457', '2026-09-05 10:19:06.713457'),
(13, 4, 2, 'Pembayaran kedua', '2026-10-30', 600000.00, 25.00, '2026-09-05 10:19:06.713457', '2026-09-05 10:19:06.713457'),
(14, 4, 3, 'Pembayaran ketiga', '2026-12-20', 800000.00, 33.33, '2026-09-05 10:19:06.713457', '2026-09-05 10:19:06.713457'),
(15, 4, 4, 'Pelunasan', '2027-02-15', 600000.00, 25.00, '2026-09-05 10:19:06.713457', '2026-09-05 10:19:06.713457'),
(16, 5, 1, 'Termin beasiswa sebelum UTS', '2026-10-20', 2000000.00, 50.00, '2026-09-05 10:19:06.713457', '2026-09-05 10:19:06.713457'),
(17, 5, 2, 'Termin beasiswa sebelum UAS', '2027-01-25', 2000000.00, 50.00, '2026-09-05 10:19:06.713457', '2026-09-05 10:19:06.713457'),
(18, 6, 1, 'Cicilan pertama', '2026-10-20', 1500000.00, 50.00, '2026-09-05 10:19:06.713457', '2026-09-05 10:19:06.713457'),
(19, 6, 2, 'Cicilan kedua', '2027-01-25', 1500000.00, 50.00, '2026-09-05 10:19:06.713457', '2026-09-05 10:19:06.713457');

-- --------------------------------------------------------

--
-- Table structure for table `charge_types`
--

CREATE TABLE `charge_types` (
  `id` bigint(20) NOT NULL,
  `code` varchar(30) NOT NULL,
  `name` varchar(100) NOT NULL,
  `description` varchar(500) DEFAULT NULL,
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `created_at` datetime(6) NOT NULL,
  `updated_at` datetime(6) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Dumping data for table `charge_types`
--

INSERT INTO `charge_types` (`id`, `code`, `name`, `description`, `is_active`, `created_at`, `updated_at`) VALUES
(1, 'SPP', 'Sumbangan Pembinaan Pendidikan', 'Biaya pendidikan utama mahasiswa untuk satu periode akademik.', 1, '2026-09-05 10:19:06.713457', '2026-09-05 10:19:06.713457'),
(2, 'UJIAN_SUSULAN', 'Ujian Susulan', 'Biaya manual untuk pelaksanaan ujian susulan mahasiswa.', 1, '2026-09-05 10:19:06.988468', '2026-09-05 10:19:06.988468');

-- --------------------------------------------------------

--
-- Table structure for table `funding_programs`
--

CREATE TABLE `funding_programs` (
  `id` bigint(20) NOT NULL,
  `code` varchar(50) NOT NULL,
  `name` varchar(150) NOT NULL,
  `program_type` varchar(30) NOT NULL,
  `funding_source` varchar(150) NOT NULL,
  `start_date` date NOT NULL,
  `end_date` date DEFAULT NULL,
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `notes` varchar(500) DEFAULT NULL,
  `created_at` datetime(6) NOT NULL,
  `updated_at` datetime(6) NOT NULL
) ;

--
-- Dumping data for table `funding_programs`
--

INSERT INTO `funding_programs` (`id`, `code`, `name`, `program_type`, `funding_source`, `start_date`, `end_date`, `is_active`, `notes`, `created_at`, `updated_at`) VALUES
(1, 'BEA-PRESTASI-100', 'Beasiswa Prestasi 100%', 'SCHOLARSHIP', 'Yayasan Kampus', '2026-08-01', '2027-01-31', 1, 'Seluruh biaya SPP dibebaskan oleh program beasiswa prestasi.', '2026-09-05 10:19:07.156482', '2026-09-05 10:19:07.156482'),
(2, 'AK-10', 'Subsidi Anak Karyawan Rp150.000', 'EMPLOYEE_SUBSIDY', 'Kampus', '2026-08-01', '2027-01-31', 1, 'Subsidi khusus untuk anak karyawan kampus.', '2026-09-05 10:19:07.156482', '2026-09-05 10:19:07.156482'),
(3, 'MKT-SPONSOR-400', 'Sponsor Marketing Rp400.000', 'MARKETING', 'Divisi Marketing', '2026-08-01', '2027-01-31', 1, 'Biaya mahasiswa dibayar langsung oleh anggaran marketing.', '2026-09-05 10:19:07.156482', '2026-09-05 10:19:07.156482');

-- --------------------------------------------------------

--
-- Table structure for table `funding_program_benefits`
--

CREATE TABLE `funding_program_benefits` (
  `id` bigint(20) NOT NULL,
  `funding_program_id` bigint(20) NOT NULL,
  `charge_type_id` bigint(20) NOT NULL,
  `benefit_type` varchar(30) NOT NULL,
  `benefit_value` decimal(18,2) NOT NULL,
  `maximum_amount` decimal(18,2) DEFAULT NULL,
  `created_at` datetime(6) NOT NULL,
  `updated_at` datetime(6) NOT NULL,
  `application_scope` varchar(20) NOT NULL DEFAULT 'PER_INVOICE',
  `maximum_uses` int(11) DEFAULT NULL,
  `priority` int(11) NOT NULL DEFAULT 100
) ;

--
-- Dumping data for table `funding_program_benefits`
--

INSERT INTO `funding_program_benefits` (`id`, `funding_program_id`, `charge_type_id`, `benefit_type`, `benefit_value`, `maximum_amount`, `created_at`, `updated_at`, `application_scope`, `maximum_uses`, `priority`) VALUES
(1, 1, 1, 'WAIVER', 100.00, 2000000.00, '2026-09-05 10:19:07.156482', '2026-11-03 12:36:13.981651', 'PER_INVOICE', NULL, 100),
(2, 2, 1, 'FIXED_AMOUNT', 150000.00, 150000.00, '2026-09-05 10:19:07.156482', '2026-11-03 12:36:13.981651', 'PER_PERIOD', 1, 100),
(3, 3, 1, 'THIRD_PARTY_PAYMENT', 400000.00, 400000.00, '2026-09-05 10:19:07.156482', '2026-11-03 12:36:13.981651', 'ONCE', 1, 100);

-- --------------------------------------------------------

--
-- Table structure for table `go_auth_sessions`
--

CREATE TABLE `go_auth_sessions` (
  `token_hash` char(64) NOT NULL,
  `user_id` varchar(255) NOT NULL,
  `csrf_token` char(64) NOT NULL,
  `security_stamp` text DEFAULT NULL,
  `expires_at` datetime(6) NOT NULL,
  `created_at` datetime(6) NOT NULL,
  `last_seen_at` datetime(6) NOT NULL,
  `remember_me` tinyint(1) NOT NULL DEFAULT 0,
  `ip_address` varchar(45) DEFAULT NULL,
  `user_agent` varchar(500) DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Dumping data for table `go_auth_sessions`
--

INSERT INTO `go_auth_sessions` (`token_hash`, `user_id`, `csrf_token`, `security_stamp`, `expires_at`, `created_at`, `last_seen_at`, `remember_me`, `ip_address`, `user_agent`) VALUES
('2297622c8d36917be370cf4e49d9ccc1eb21482d7864643f6eb29b746eef5526', '54434c08-936f-4e54-ab0b-565319c28f6f', 'QZ23YRmmHMu46y46B9OBiaasNEMoviEtfrQ3eKn-_yQ', 'RMBM6GmQIFqy0dVy1ISOjHuMZEdkceCeaRKrgj9WOlk', '2026-11-03 21:39:36.538232', '2026-11-03 13:39:36.538232', '2026-11-03 13:39:36.614868', 0, '127.0.0.1', 'Mozilla/5.0 (Windows NT; Windows NT 10.0; en-US) WindowsPowerShell/5.1.26100.9818'),
('31de04b497a0ea2e0b95f6b1f377ab311c659c0d5c97800c22447ea11df9aceb', '54434c08-936f-4e54-ab0b-565319c28f6f', '8g-ltotHEZCl7yJDuGL8P_a9K24WZzdZpS_AkKlpwGE', 'RMBM6GmQIFqy0dVy1ISOjHuMZEdkceCeaRKrgj9WOlk', '2026-11-03 21:48:51.277948', '2026-11-03 13:48:51.277948', '2026-11-03 13:48:51.356267', 0, '127.0.0.1', 'Mozilla/5.0 (Windows NT; Windows NT 10.0; en-US) WindowsPowerShell/5.1.26100.9818'),
('68fc21177fe9acd349b53826abc25fdc0b0739c4c81c4212fa0cd2e956d02255', '54434c08-936f-4e54-ab0b-565319c28f6f', 'UjWI6b6Bk7FbqX3ifaokLc0ZxmlfGceXPgmkTvmLF3Y', 'RMBM6GmQIFqy0dVy1ISOjHuMZEdkceCeaRKrgj9WOlk', '2026-11-03 22:01:04.951778', '2026-11-03 14:01:04.951778', '2026-11-03 14:01:05.052709', 0, '127.0.0.1', 'Mozilla/5.0 (Windows NT; Windows NT 10.0; en-US) WindowsPowerShell/5.1.26100.9818'),
('9271f527b8bc0bb4b703f8ac59c24225c3f4dcaf29a03d6614a3a7e669dbc196', '54434c08-936f-4e54-ab0b-565319c28f6f', 'iLLnCNJwXIdA_fY_NnvkfaIVpG9f7Y12-bNVJaIpXvw', 'RMBM6GmQIFqy0dVy1ISOjHuMZEdkceCeaRKrgj9WOlk', '2026-11-03 22:08:36.899509', '2026-11-03 14:08:36.899509', '2026-11-03 14:08:37.153030', 0, '127.0.0.1', 'Mozilla/5.0 (Windows NT; Windows NT 10.0; en-US) WindowsPowerShell/5.1.26100.9818'),
('9ee57d931dd5499eba762f928fc4f7a9a5c9a50ec63b589bd36a66adca016175', '54434c08-936f-4e54-ab0b-565319c28f6f', '3RXZyOwhz5BDlLGqD-grclWeJZSVuP6MQGYVjlpAawY', 'RMBM6GmQIFqy0dVy1ISOjHuMZEdkceCeaRKrgj9WOlk', '2026-11-03 22:08:51.519993', '2026-11-03 14:08:51.519993', '2026-11-03 14:08:51.660142', 0, '127.0.0.1', 'Mozilla/5.0 (Windows NT; Windows NT 10.0; en-US) WindowsPowerShell/5.1.26100.9818'),
('b1d4e8fc4ee0112be73ec2a1548962597b6b9a3f05c307c057f9a42e6f4fcba8', '54434c08-936f-4e54-ab0b-565319c28f6f', 'Wu_42WS-Qh9Xiuj8hlyqe8OfVGb53I6DDFZ3lFY-GJQ', 'RMBM6GmQIFqy0dVy1ISOjHuMZEdkceCeaRKrgj9WOlk', '2026-11-03 21:33:46.704325', '2026-11-03 13:33:46.704325', '2026-11-03 14:17:46.120068', 0, '::1', 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/156.0.0.0 Safari/537.36'),
('b6bf9e6755cb710f1004dcf4715139847bd7b556980f61e5d1e08cd610f0e07f', '54434c08-936f-4e54-ab0b-565319c28f6f', 'bVc9zmrVibuYohUzOuVwqINTybf3ZkMUm2dvjZeVIvU', 'RMBM6GmQIFqy0dVy1ISOjHuMZEdkceCeaRKrgj9WOlk', '2026-11-03 21:43:12.751582', '2026-11-03 13:43:12.751582', '2026-11-03 13:43:12.825585', 0, '127.0.0.1', 'Mozilla/5.0 (Windows NT; Windows NT 10.0; en-US) WindowsPowerShell/5.1.26100.9818'),
('e0b642521435b4e5bf90a70556379b1b1c3ced3affb62941c97d6dbe37cb68ac', '54434c08-936f-4e54-ab0b-565319c28f6f', 'lCPKuq84FCXQgOUdIhUYxI5qpfn3jNgOqO7Ko_NRbf4', 'RMBM6GmQIFqy0dVy1ISOjHuMZEdkceCeaRKrgj9WOlk', '2026-11-03 22:08:46.289661', '2026-11-03 14:08:46.289661', '2026-11-03 14:08:46.748215', 0, '127.0.0.1', 'Mozilla/5.0 (Windows NT; Windows NT 10.0; en-US) WindowsPowerShell/5.1.26100.9818');

-- --------------------------------------------------------

--
-- Table structure for table `go_password_reset_tokens`
--

CREATE TABLE `go_password_reset_tokens` (
  `token_hash` char(64) NOT NULL,
  `user_id` varchar(255) NOT NULL,
  `expires_at` datetime(6) NOT NULL,
  `created_at` datetime(6) NOT NULL,
  `used_at` datetime(6) DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- Table structure for table `invoice_generation_batches`
--

CREATE TABLE `invoice_generation_batches` (
  `id` bigint(20) NOT NULL,
  `batch_number` varchar(50) NOT NULL,
  `academic_period_id` bigint(20) DEFAULT NULL,
  `status` varchar(20) NOT NULL,
  `total_candidates` int(11) NOT NULL,
  `total_created` int(11) NOT NULL,
  `total_skipped` int(11) NOT NULL,
  `total_failed` int(11) NOT NULL,
  `total_amount` decimal(18,2) NOT NULL,
  `notes` varchar(500) DEFAULT NULL,
  `initiated_by_id` varchar(255) DEFAULT NULL,
  `started_at` datetime(6) NOT NULL,
  `completed_at` datetime(6) DEFAULT NULL,
  `cancellation_reason` varchar(500) DEFAULT NULL,
  `cancelled_by_id` varchar(255) DEFAULT NULL,
  `cancelled_at` datetime(6) DEFAULT NULL
) ;

-- --------------------------------------------------------

--
-- Table structure for table `invoice_generation_batch_items`
--

CREATE TABLE `invoice_generation_batch_items` (
  `id` bigint(20) NOT NULL,
  `invoice_generation_batch_id` bigint(20) NOT NULL,
  `student_billing_policy_id` bigint(20) NOT NULL,
  `billing_policy_schedule_id` bigint(20) NOT NULL,
  `student_invoice_id` bigint(20) DEFAULT NULL,
  `status` varchar(20) NOT NULL,
  `proposed_amount` decimal(18,2) NOT NULL,
  `message` varchar(500) DEFAULT NULL,
  `created_at` datetime(6) NOT NULL
) ;

-- --------------------------------------------------------

--
-- Table structure for table `payments`
--

CREATE TABLE `payments` (
  `id` bigint(20) NOT NULL,
  `payment_number` varchar(50) NOT NULL,
  `student_id` bigint(20) NOT NULL,
  `payment_date` datetime(6) NOT NULL,
  `amount` decimal(18,2) NOT NULL,
  `payment_method` varchar(30) NOT NULL,
  `reference_number` varchar(100) DEFAULT NULL,
  `status` varchar(20) NOT NULL,
  `received_by_id` varchar(255) NOT NULL,
  `verified_by_id` varchar(255) DEFAULT NULL,
  `verified_at` datetime(6) DEFAULT NULL,
  `notes` varchar(500) DEFAULT NULL,
  `created_at` datetime(6) NOT NULL,
  `updated_at` datetime(6) NOT NULL,
  `funding_program_id` bigint(20) DEFAULT NULL,
  `payer_name` varchar(150) DEFAULT NULL,
  `payer_type` varchar(30) NOT NULL DEFAULT 'STUDENT',
  `cancelled_at` datetime(6) DEFAULT NULL,
  `cancelled_by_id` varchar(255) DEFAULT NULL,
  `cancelled_reason` varchar(500) DEFAULT NULL,
  `rejected_reason` varchar(500) DEFAULT NULL,
  `proof_stored_file_name` varchar(100) DEFAULT NULL
) ;

--
-- Dumping data for table `payments`
--

INSERT INTO `payments` (`id`, `payment_number`, `student_id`, `payment_date`, `amount`, `payment_method`, `reference_number`, `status`, `received_by_id`, `verified_by_id`, `verified_at`, `notes`, `created_at`, `updated_at`, `funding_program_id`, `payer_name`, `payer_type`, `cancelled_at`, `cancelled_by_id`, `cancelled_reason`, `rejected_reason`, `proof_stored_file_name`) VALUES
(1, 'PAY-2026-0001', 1, '2026-09-10 09:15:00.000000', 600000.00, 'BANK_TRANSFER', 'BCA-TRX-20260910-001', 'VERIFIED', 'e5010e83-353e-4d06-8589-fd9245750e39', '54434c08-936f-4e54-ab0b-565319c28f6f', '2026-09-10 10:00:00.000000', 'Satu pembayaran dialokasikan ke dua tagihan SPP.', '2026-09-05 10:19:07.444030', '2026-09-05 10:19:07.444030', NULL, 'Ahmad Fauzan', 'STUDENT', NULL, NULL, NULL, NULL, NULL),
(2, 'PAY-2026-0002', 1, '2026-10-05 13:30:00.000000', 100000.00, 'QRIS', 'QRIS-20261005-002', 'VERIFIED', 'e5010e83-353e-4d06-8589-fd9245750e39', '54434c08-936f-4e54-ab0b-565319c28f6f', '2026-10-05 13:45:00.000000', 'Pembayaran kedua untuk tagihan SPP Oktober.', '2026-09-05 10:19:07.444030', '2026-09-05 10:19:07.444030', NULL, 'Orang Tua Ahmad Fauzan', 'PARENT', NULL, NULL, NULL, NULL, NULL),
(3, 'PAY-2026-0003', 3, '2026-09-04 08:20:00.000000', 275000.00, 'CASH', NULL, 'PENDING', 'e5010e83-353e-4d06-8589-fd9245750e39', NULL, NULL, 'Pembayaran tunai ujian susulan menunggu verifikasi.', '2026-09-05 10:19:07.444030', '2026-09-05 10:19:07.444030', NULL, 'Citra Lestari', 'STUDENT', NULL, NULL, NULL, NULL, NULL),
(4, 'PAY-2026-0004', 4, '2026-09-03 15:00:00.000000', 400000.00, 'BANK_TRANSFER', 'BNI-TRX-20260903-004', 'REJECTED', 'e5010e83-353e-4d06-8589-fd9245750e39', '54434c08-936f-4e54-ab0b-565319c28f6f', '2026-09-03 16:00:00.000000', 'Ditolak karena bukti transfer tidak valid; tidak menambah pembayaran tagihan.', '2026-09-05 10:19:07.444030', '2026-09-05 10:19:07.444030', NULL, 'Dewi Anggraini', 'STUDENT', NULL, NULL, NULL, NULL, NULL),
(5, 'PAY-2026-0005', 4, '2026-09-05 10:00:00.000000', 400000.00, 'BANK_TRANSFER', 'MKT-BUDGET-2026-004', 'VERIFIED', 'e5010e83-353e-4d06-8589-fd9245750e39', '54434c08-936f-4e54-ab0b-565319c28f6f', '2026-09-05 10:15:00.000000', 'Contoh pembayaran pihak ketiga: Divisi Marketing membayar tagihan mahasiswa.', '2026-09-05 10:19:07.444030', '2026-09-05 10:19:07.444030', 3, 'Divisi Marketing Kampus', 'MARKETING', NULL, NULL, NULL, NULL, NULL);

-- --------------------------------------------------------

--
-- Table structure for table `payment_allocations`
--

CREATE TABLE `payment_allocations` (
  `id` bigint(20) NOT NULL,
  `payment_id` bigint(20) NOT NULL,
  `student_invoice_id` bigint(20) NOT NULL,
  `amount` decimal(18,2) NOT NULL,
  `created_at` datetime(6) NOT NULL
) ;

--
-- Dumping data for table `payment_allocations`
--

INSERT INTO `payment_allocations` (`id`, `payment_id`, `student_invoice_id`, `amount`, `created_at`) VALUES
(1, 1, 1, 500000.00, '2026-09-05 10:19:07.444030'),
(2, 1, 2, 100000.00, '2026-09-05 10:19:07.444030'),
(3, 2, 2, 100000.00, '2026-09-05 10:19:07.444030'),
(4, 3, 7, 275000.00, '2026-09-05 10:19:07.444030'),
(5, 4, 5, 400000.00, '2026-09-05 10:19:07.444030'),
(6, 5, 5, 400000.00, '2026-09-05 10:19:07.444030');

-- --------------------------------------------------------

--
-- Table structure for table `permissions`
--

CREATE TABLE `permissions` (
  `id` bigint(20) NOT NULL,
  `name` varchar(255) NOT NULL,
  `guard_name` varchar(255) NOT NULL,
  `created_at` datetime(6) NOT NULL,
  `updated_at` datetime(6) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Dumping data for table `permissions`
--

INSERT INTO `permissions` (`id`, `name`, `guard_name`, `created_at`, `updated_at`) VALUES
(1, 'dashboard.super-admin.view', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(2, 'dashboard.finance.view', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(3, 'dashboard.executive.view', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(4, 'dashboard.academic.view', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(5, 'dashboard.marketing.view', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(6, 'users.view', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(7, 'users.create', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(8, 'users.update', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(9, 'users.reset-password', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(10, 'users.manage-roles', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(11, 'roles.view', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(12, 'roles.manage', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(13, 'permissions.view', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(14, 'permissions.manage', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(15, 'students.view', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(16, 'students.create', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(17, 'students.update', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(18, 'payment-types.view', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(19, 'payment-types.manage', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(20, 'payment-schemes.view', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(21, 'payment-schemes.manage', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(22, 'bills.view', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(23, 'bills.create', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(24, 'bills.generate', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(25, 'bills.update', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(26, 'bills.cancel', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(27, 'payments.view', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(28, 'payments.create', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(29, 'payments.update', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(30, 'payments.verify', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(31, 'payments.reject', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(32, 'payments.cancel', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(33, 'funding-programs.view', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(34, 'funding-programs.create', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(35, 'funding-programs.update', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(36, 'funding-programs.delete', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(37, 'student-funding-programs.view', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(38, 'student-funding-programs.create', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(39, 'student-funding-programs.update', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(40, 'student-funding-programs.delete', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(41, 'student-funding-programs.generate', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(42, 'student-invoice-adjustments.view', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(43, 'student-invoice-adjustments.create', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(44, 'student-invoice-adjustments.update', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(45, 'student-invoice-adjustments.delete', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(46, 'receipts.view', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(47, 'receipts.print', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(48, 'discounts.view', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(49, 'discounts.create', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(50, 'discounts.approve', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(51, 'discounts.reject', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(52, 'dispensations.view', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(53, 'dispensations.create', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(54, 'dispensations.approve', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(55, 'dispensations.reject', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(56, 'reports.operational.view', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(57, 'reports.financial.view', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(58, 'reports.executive.view', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(59, 'reports.export', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(60, 'academic-status.view', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(61, 'marketing-students.view', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(62, 'audit-logs.view', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(63, 'login-histories.view', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(64, 'settings.view', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(65, 'settings.update', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(66, 'backups.create', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(67, 'study-programs.view', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(68, 'study-programs.create', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(69, 'study-programs.update', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(70, 'study-programs.delete', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(71, 'academic-periods.view', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(72, 'academic-periods.create', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(73, 'academic-periods.update', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(74, 'academic-periods.delete', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(75, 'student-categories.view', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(76, 'student-categories.create', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(77, 'student-categories.update', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(78, 'student-categories.delete', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(79, 'charge-types.view', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(80, 'charge-types.create', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(81, 'charge-types.update', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(82, 'charge-types.delete', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(83, 'students.delete', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(84, 'billing-policies.view', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(85, 'billing-policies.create', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(86, 'billing-policies.update', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(87, 'billing-policies.delete', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(88, 'billing-policy-schedules.view', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(89, 'billing-policy-schedules.create', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(90, 'billing-policy-schedules.update', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(91, 'billing-policy-schedules.delete', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(92, 'student-billing-policies.view', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(93, 'student-billing-policies.create', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(94, 'student-billing-policies.update', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(95, 'student-billing-policies.delete', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(96, 'student-invoices.view', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(97, 'student-invoices.create', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(98, 'student-invoices.generate', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(99, 'student-invoices.update', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(100, 'student-invoices.delete', 'web', '2026-09-05 10:19:06.135186', '2026-09-05 10:19:06.135186'),
(101, 'invoice-generation-batches.view', 'web', '2026-09-05 13:08:08.995482', '2026-09-05 13:08:08.995482'),
(102, 'invoice-generation-batches.cancel', 'web', '2026-09-05 13:08:08.995482', '2026-09-05 13:08:08.995482'),
(103, 'student-billing-policy-batches.view', 'web', '2026-09-05 13:41:38.214086', '2026-09-05 13:41:38.214086'),
(104, 'student-billing-policy-batches.create', 'web', '2026-09-05 13:41:38.214086', '2026-09-05 13:41:38.214086'),
(105, 'student-billing-policy-batches.cancel', 'web', '2026-09-05 13:41:38.214086', '2026-09-05 13:41:38.214086');

-- --------------------------------------------------------

--
-- Table structure for table `roles`
--

CREATE TABLE `roles` (
  `Id` varchar(255) NOT NULL,
  `Name` varchar(256) DEFAULT NULL,
  `NormalizedName` varchar(256) DEFAULT NULL,
  `ConcurrencyStamp` longtext DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Dumping data for table `roles`
--

INSERT INTO `roles` (`Id`, `Name`, `NormalizedName`, `ConcurrencyStamp`) VALUES
('341c5cd7-ce65-4df7-b4a7-b9ae302558cd', 'Staff Akademik', 'STAFF AKADEMIK', 'b60f541f-9bc2-4be0-ab5c-1275a69797e9'),
('356113ab-ea7b-45b8-b082-0f3882aec232', 'Marketing / PMB', 'MARKETING / PMB', '431513c4-9aa8-4f33-91ae-db7e1899d6cb'),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 'Admin Keuangan', 'ADMIN KEUANGAN', '8a334bf0-df43-4c9c-8a14-ecb917fc82ea'),
('915976e6-3b8a-4866-8f81-d3d4dd533ba2', 'Rektor', 'REKTOR', 'fbefe9fc-a421-4229-9b0c-f26aac85f3da'),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 'Super Admin', 'SUPER ADMIN', '2675c332-a263-4874-b346-aacd4f9f4ec9'),
('f04f98fb-db01-41b5-9865-c020f16e7ae4', 'Manager Keuangan', 'MANAGER KEUANGAN', 'ad1555f4-3e07-4d9e-9713-c7d686d9495d');

-- --------------------------------------------------------

--
-- Table structure for table `role_claims`
--

CREATE TABLE `role_claims` (
  `Id` int(11) NOT NULL,
  `RoleId` varchar(255) NOT NULL,
  `ClaimType` longtext DEFAULT NULL,
  `ClaimValue` longtext DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- Table structure for table `role_permissions`
--

CREATE TABLE `role_permissions` (
  `role_id` varchar(255) NOT NULL,
  `permission_id` bigint(20) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Dumping data for table `role_permissions`
--

INSERT INTO `role_permissions` (`role_id`, `permission_id`) VALUES
('341c5cd7-ce65-4df7-b4a7-b9ae302558cd', 4),
('341c5cd7-ce65-4df7-b4a7-b9ae302558cd', 15),
('341c5cd7-ce65-4df7-b4a7-b9ae302558cd', 16),
('341c5cd7-ce65-4df7-b4a7-b9ae302558cd', 17),
('341c5cd7-ce65-4df7-b4a7-b9ae302558cd', 22),
('341c5cd7-ce65-4df7-b4a7-b9ae302558cd', 52),
('341c5cd7-ce65-4df7-b4a7-b9ae302558cd', 60),
('341c5cd7-ce65-4df7-b4a7-b9ae302558cd', 67),
('341c5cd7-ce65-4df7-b4a7-b9ae302558cd', 68),
('341c5cd7-ce65-4df7-b4a7-b9ae302558cd', 69),
('341c5cd7-ce65-4df7-b4a7-b9ae302558cd', 71),
('341c5cd7-ce65-4df7-b4a7-b9ae302558cd', 72),
('341c5cd7-ce65-4df7-b4a7-b9ae302558cd', 73),
('341c5cd7-ce65-4df7-b4a7-b9ae302558cd', 75),
('341c5cd7-ce65-4df7-b4a7-b9ae302558cd', 76),
('341c5cd7-ce65-4df7-b4a7-b9ae302558cd', 77),
('341c5cd7-ce65-4df7-b4a7-b9ae302558cd', 79),
('341c5cd7-ce65-4df7-b4a7-b9ae302558cd', 84),
('341c5cd7-ce65-4df7-b4a7-b9ae302558cd', 88),
('341c5cd7-ce65-4df7-b4a7-b9ae302558cd', 92),
('341c5cd7-ce65-4df7-b4a7-b9ae302558cd', 96),
('356113ab-ea7b-45b8-b082-0f3882aec232', 5),
('356113ab-ea7b-45b8-b082-0f3882aec232', 15),
('356113ab-ea7b-45b8-b082-0f3882aec232', 48),
('356113ab-ea7b-45b8-b082-0f3882aec232', 49),
('356113ab-ea7b-45b8-b082-0f3882aec232', 61),
('356113ab-ea7b-45b8-b082-0f3882aec232', 67),
('356113ab-ea7b-45b8-b082-0f3882aec232', 75),
('356113ab-ea7b-45b8-b082-0f3882aec232', 84),
('356113ab-ea7b-45b8-b082-0f3882aec232', 92),
('356113ab-ea7b-45b8-b082-0f3882aec232', 96),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 2),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 15),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 16),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 17),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 18),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 19),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 20),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 21),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 22),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 23),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 24),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 25),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 27),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 28),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 29),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 30),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 31),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 33),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 34),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 35),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 37),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 38),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 39),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 41),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 42),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 43),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 44),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 46),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 47),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 48),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 49),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 52),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 53),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 56),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 59),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 62),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 67),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 71),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 75),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 79),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 80),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 81),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 84),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 85),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 86),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 87),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 88),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 89),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 90),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 91),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 92),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 93),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 94),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 95),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 96),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 97),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 98),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 99),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 100),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 101),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 103),
('50c7e970-09cc-44b2-9ef7-3eb9566620d6', 104),
('915976e6-3b8a-4866-8f81-d3d4dd533ba2', 3),
('915976e6-3b8a-4866-8f81-d3d4dd533ba2', 48),
('915976e6-3b8a-4866-8f81-d3d4dd533ba2', 50),
('915976e6-3b8a-4866-8f81-d3d4dd533ba2', 51),
('915976e6-3b8a-4866-8f81-d3d4dd533ba2', 52),
('915976e6-3b8a-4866-8f81-d3d4dd533ba2', 54),
('915976e6-3b8a-4866-8f81-d3d4dd533ba2', 55),
('915976e6-3b8a-4866-8f81-d3d4dd533ba2', 57),
('915976e6-3b8a-4866-8f81-d3d4dd533ba2', 58),
('915976e6-3b8a-4866-8f81-d3d4dd533ba2', 59),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 1),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 2),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 3),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 4),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 5),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 6),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 7),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 8),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 9),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 10),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 11),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 12),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 13),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 14),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 15),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 16),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 17),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 18),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 19),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 20),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 21),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 22),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 23),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 24),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 25),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 26),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 27),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 28),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 29),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 30),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 31),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 32),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 33),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 34),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 35),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 36),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 37),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 38),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 39),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 40),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 41),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 42),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 43),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 44),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 45),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 46),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 47),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 48),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 49),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 50),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 51),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 52),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 53),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 54),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 55),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 56),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 57),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 58),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 59),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 60),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 61),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 62),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 63),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 64),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 65),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 66),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 67),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 68),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 69),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 70),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 71),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 72),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 73),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 74),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 75),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 76),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 77),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 78),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 79),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 80),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 81),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 82),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 83),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 84),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 85),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 86),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 87),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 88),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 89),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 90),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 91),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 92),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 93),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 94),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 95),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 96),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 97),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 98),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 99),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 100),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 101),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 102),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 103),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 104),
('bdfd09a6-f570-4ae8-97f9-00f9723cc123', 105),
('f04f98fb-db01-41b5-9865-c020f16e7ae4', 2),
('f04f98fb-db01-41b5-9865-c020f16e7ae4', 15),
('f04f98fb-db01-41b5-9865-c020f16e7ae4', 22),
('f04f98fb-db01-41b5-9865-c020f16e7ae4', 27),
('f04f98fb-db01-41b5-9865-c020f16e7ae4', 32),
('f04f98fb-db01-41b5-9865-c020f16e7ae4', 46),
('f04f98fb-db01-41b5-9865-c020f16e7ae4', 47),
('f04f98fb-db01-41b5-9865-c020f16e7ae4', 48),
('f04f98fb-db01-41b5-9865-c020f16e7ae4', 50),
('f04f98fb-db01-41b5-9865-c020f16e7ae4', 51),
('f04f98fb-db01-41b5-9865-c020f16e7ae4', 52),
('f04f98fb-db01-41b5-9865-c020f16e7ae4', 54),
('f04f98fb-db01-41b5-9865-c020f16e7ae4', 55),
('f04f98fb-db01-41b5-9865-c020f16e7ae4', 56),
('f04f98fb-db01-41b5-9865-c020f16e7ae4', 57),
('f04f98fb-db01-41b5-9865-c020f16e7ae4', 59),
('f04f98fb-db01-41b5-9865-c020f16e7ae4', 62),
('f04f98fb-db01-41b5-9865-c020f16e7ae4', 84),
('f04f98fb-db01-41b5-9865-c020f16e7ae4', 88),
('f04f98fb-db01-41b5-9865-c020f16e7ae4', 92),
('f04f98fb-db01-41b5-9865-c020f16e7ae4', 96),
('f04f98fb-db01-41b5-9865-c020f16e7ae4', 99),
('f04f98fb-db01-41b5-9865-c020f16e7ae4', 101),
('f04f98fb-db01-41b5-9865-c020f16e7ae4', 102),
('f04f98fb-db01-41b5-9865-c020f16e7ae4', 103),
('f04f98fb-db01-41b5-9865-c020f16e7ae4', 105);

-- --------------------------------------------------------

--
-- Table structure for table `students`
--

CREATE TABLE `students` (
  `id` bigint(20) NOT NULL,
  `nim` varchar(30) NOT NULL,
  `name` varchar(150) NOT NULL,
  `study_program_id` bigint(20) NOT NULL,
  `cohort_year` int(11) NOT NULL,
  `status` varchar(20) NOT NULL DEFAULT 'ACTIVE',
  `email` varchar(150) DEFAULT NULL,
  `phone_number` varchar(30) DEFAULT NULL,
  `created_at` datetime(6) NOT NULL,
  `updated_at` datetime(6) NOT NULL,
  `student_category_id` bigint(20) DEFAULT NULL,
  `cohort_year_class` varchar(30) DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Dumping data for table `students`
--

INSERT INTO `students` (`id`, `nim`, `name`, `study_program_id`, `cohort_year`, `status`, `email`, `phone_number`, `created_at`, `updated_at`, `student_category_id`, `cohort_year_class`) VALUES
(1, '20260001', 'Ahmad Fauzan', 1, 2026, 'ACTIVE', 'ahmad.fauzan@mahasiswa.siato.local', '081234567801', '2026-09-05 10:19:06.571956', '2026-09-05 10:19:06.571956', 1, '2026.1.A'),
(2, '20260002', 'Siti Aisyah', 2, 2026, 'ACTIVE', 'siti.aisyah@mahasiswa.siato.local', '081234567802', '2026-09-05 10:19:06.571956', '2026-09-05 10:19:06.571956', 2, '2026.1.B'),
(3, '20250003', 'Budi Santoso', 1, 2025, 'ACTIVE', 'budi.santoso@mahasiswa.siato.local', '081234567803', '2026-09-05 10:19:06.571956', '2026-09-05 10:19:06.571956', 3, '2025.1.A'),
(4, '20250004', 'Dewi Lestari', 2, 2025, 'ACTIVE', 'dewi.lestari@mahasiswa.siato.local', '081234567804', '2026-09-05 10:19:06.571956', '2026-09-05 10:19:06.571956', 4, '2025.1.B'),
(5, '20240005', 'Rizky Pratama', 1, 2024, 'ACTIVE', 'rizky.pratama@mahasiswa.siato.local', '081234567805', '2026-09-05 10:19:06.571956', '2026-09-05 10:19:06.571956', 1, '2024.1.A');

-- --------------------------------------------------------

--
-- Table structure for table `student_billing_policies`
--

CREATE TABLE `student_billing_policies` (
  `id` bigint(20) NOT NULL,
  `student_id` bigint(20) NOT NULL,
  `billing_policy_id` bigint(20) NOT NULL,
  `effective_amount` decimal(18,2) NOT NULL,
  `assignment_type` varchar(20) NOT NULL,
  `status` varchar(20) NOT NULL,
  `approved_by_id` varchar(255) DEFAULT NULL,
  `assigned_at` datetime(6) NOT NULL,
  `notes` varchar(500) DEFAULT NULL,
  `created_at` datetime(6) NOT NULL,
  `updated_at` datetime(6) NOT NULL
) ;

--
-- Dumping data for table `student_billing_policies`
--

INSERT INTO `student_billing_policies` (`id`, `student_id`, `billing_policy_id`, `effective_amount`, `assignment_type`, `status`, `approved_by_id`, `assigned_at`, `notes`, `created_at`, `updated_at`) VALUES
(1, 1, 1, 3000000.00, 'AUTOMATIC', 'ACTIVE', NULL, '2026-09-01 08:00:00.000000', 'Tarif otomatis berdasarkan TI, reguler, dan kelas 2026.1.A.', '2026-09-05 10:19:06.713457', '2026-09-05 10:19:06.713457'),
(2, 2, 5, 0.00, 'MANUAL', 'ACTIVE', '54434c08-936f-4e54-ab0b-565319c28f6f', '2026-09-01 08:00:00.000000', 'Beasiswa 100%; tarif dasar ditanggung oleh program beasiswa.', '2026-09-05 10:19:06.713457', '2026-09-05 10:19:06.713457'),
(3, 3, 3, 1500000.00, 'AUTOMATIC', 'ACTIVE', NULL, '2026-09-01 08:00:00.000000', 'Tarif khusus golongan anak karyawan.', '2026-09-05 10:19:06.713457', '2026-09-05 10:19:06.713457'),
(4, 4, 4, 2400000.00, 'MANUAL', 'ACTIVE', '54434c08-936f-4e54-ab0b-565319c28f6f', '2026-09-01 08:00:00.000000', 'Penetapan manual sesuai persetujuan marketing.', '2026-09-05 10:19:06.713457', '2026-09-05 10:19:06.713457'),
(5, 5, 6, 3000000.00, 'AUTOMATIC', 'ACTIVE', NULL, '2026-09-01 08:00:00.000000', 'Tarif otomatis untuk mahasiswa reguler TI angkatan 2024.', '2026-09-05 10:19:06.713457', '2026-09-05 10:19:06.713457');

-- --------------------------------------------------------

--
-- Table structure for table `student_billing_policy_batches`
--

CREATE TABLE `student_billing_policy_batches` (
  `id` bigint(20) NOT NULL,
  `batch_number` varchar(50) NOT NULL,
  `billing_policy_id` bigint(20) NOT NULL,
  `study_program_id` bigint(20) DEFAULT NULL,
  `cohort_year` int(11) DEFAULT NULL,
  `student_category_id` bigint(20) DEFAULT NULL,
  `cohort_year_class` varchar(30) DEFAULT NULL,
  `status` varchar(20) NOT NULL,
  `total_candidates` int(11) NOT NULL,
  `total_created` int(11) NOT NULL,
  `total_skipped` int(11) NOT NULL,
  `total_failed` int(11) NOT NULL,
  `initiated_by_id` varchar(255) DEFAULT NULL,
  `created_at` datetime(6) NOT NULL,
  `cancellation_reason` varchar(500) DEFAULT NULL,
  `cancelled_by_id` varchar(255) DEFAULT NULL,
  `cancelled_at` datetime(6) DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- Table structure for table `student_billing_policy_batch_items`
--

CREATE TABLE `student_billing_policy_batch_items` (
  `id` bigint(20) NOT NULL,
  `student_billing_policy_batch_id` bigint(20) NOT NULL,
  `student_id` bigint(20) NOT NULL,
  `student_billing_policy_id` bigint(20) DEFAULT NULL,
  `effective_amount` decimal(18,2) NOT NULL,
  `status` varchar(20) NOT NULL,
  `message` varchar(500) DEFAULT NULL,
  `created_at` datetime(6) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- Table structure for table `student_categories`
--

CREATE TABLE `student_categories` (
  `id` bigint(20) NOT NULL,
  `code` varchar(30) NOT NULL,
  `name` varchar(100) NOT NULL,
  `description` varchar(500) DEFAULT NULL,
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `created_at` datetime(6) NOT NULL,
  `updated_at` datetime(6) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Dumping data for table `student_categories`
--

INSERT INTO `student_categories` (`id`, `code`, `name`, `description`, `is_active`, `created_at`, `updated_at`) VALUES
(1, 'REGULER', 'Mahasiswa Reguler', 'Mahasiswa dengan tarif dan ketentuan pembayaran reguler kampus.', 1, '2026-09-05 10:19:06.571956', '2026-09-05 10:19:06.571956'),
(2, 'BEASISWA', 'Mahasiswa Beasiswa', 'Mahasiswa yang memperoleh pembiayaan atau potongan melalui program beasiswa.', 1, '2026-09-05 10:19:06.571956', '2026-09-05 10:19:06.571956'),
(3, 'ANAK_KARYAWAN', 'Anak Karyawan', 'Mahasiswa yang memperoleh kebijakan biaya khusus sebagai anak karyawan kampus.', 1, '2026-09-05 10:19:06.571956', '2026-09-05 10:19:06.571956'),
(4, 'KHUSUS_MARKETING', 'Khusus Marketing', 'Mahasiswa dengan kebijakan pembayaran khusus yang disetujui bagian marketing kampus.', 1, '2026-09-05 10:19:06.571956', '2026-09-05 10:19:06.571956');

-- --------------------------------------------------------

--
-- Table structure for table `student_funding_programs`
--

CREATE TABLE `student_funding_programs` (
  `id` bigint(20) NOT NULL,
  `student_id` bigint(20) NOT NULL,
  `funding_program_id` bigint(20) NOT NULL,
  `academic_period_id` bigint(20) DEFAULT NULL,
  `reference_number` varchar(100) DEFAULT NULL,
  `start_date` date NOT NULL,
  `end_date` date DEFAULT NULL,
  `status` varchar(20) NOT NULL,
  `notes` varchar(500) DEFAULT NULL,
  `created_at` datetime(6) NOT NULL,
  `updated_at` datetime(6) NOT NULL
) ;

--
-- Dumping data for table `student_funding_programs`
--

INSERT INTO `student_funding_programs` (`id`, `student_id`, `funding_program_id`, `academic_period_id`, `reference_number`, `start_date`, `end_date`, `status`, `notes`, `created_at`, `updated_at`) VALUES
(1, 2, 1, 1, 'SK-BEA-2026-001', '2026-08-01', '2027-01-31', 'ACTIVE', 'Mahasiswa menerima pembebasan SPP 100%.', '2026-09-05 10:19:07.156482', '2026-09-05 10:19:07.156482'),
(2, 5, 2, 1, 'SK-SDM-2026-005', '2026-08-01', '2027-01-31', 'ACTIVE', 'Mahasiswa menerima subsidi sebagai anak karyawan.', '2026-09-05 10:19:07.156482', '2026-09-05 10:19:07.156482'),
(3, 4, 3, 1, 'MKT-2026-004', '2026-08-01', '2027-01-31', 'ACTIVE', 'Tagihan dibayar oleh Divisi Marketing.', '2026-09-05 10:19:07.156482', '2026-09-05 10:19:07.156482');

-- --------------------------------------------------------

--
-- Table structure for table `student_invoices`
--

CREATE TABLE `student_invoices` (
  `id` bigint(20) NOT NULL,
  `invoice_number` varchar(50) NOT NULL,
  `student_id` bigint(20) NOT NULL,
  `academic_period_id` bigint(20) DEFAULT NULL,
  `charge_type_id` bigint(20) NOT NULL,
  `student_billing_policy_id` bigint(20) DEFAULT NULL,
  `billing_policy_schedule_id` bigint(20) DEFAULT NULL,
  `sequence_no` int(11) DEFAULT NULL,
  `description` varchar(255) NOT NULL,
  `original_amount` decimal(18,2) NOT NULL,
  `discount_amount` decimal(18,2) NOT NULL DEFAULT 0.00,
  `penalty_amount` decimal(18,2) NOT NULL DEFAULT 0.00,
  `final_amount` decimal(18,2) NOT NULL,
  `paid_amount` decimal(18,2) NOT NULL DEFAULT 0.00,
  `due_date` date NOT NULL,
  `status` varchar(20) NOT NULL,
  `source_type` varchar(20) NOT NULL,
  `notes` varchar(500) DEFAULT NULL,
  `created_at` datetime(6) NOT NULL,
  `updated_at` datetime(6) NOT NULL,
  `version` bigint(20) NOT NULL DEFAULT 1,
  `invoice_generation_batch_id` bigint(20) DEFAULT NULL
) ;

--
-- Dumping data for table `student_invoices`
--

INSERT INTO `student_invoices` (`id`, `invoice_number`, `student_id`, `academic_period_id`, `charge_type_id`, `student_billing_policy_id`, `billing_policy_schedule_id`, `sequence_no`, `description`, `original_amount`, `discount_amount`, `penalty_amount`, `final_amount`, `paid_amount`, `due_date`, `status`, `source_type`, `notes`, `created_at`, `updated_at`, `version`, `invoice_generation_batch_id`) VALUES
(1, 'INV-2026-0001', 1, 1, 1, 1, 1, 1, 'Cicilan September', 500000.00, 0.00, 0.00, 500000.00, 500000.00, '2026-09-10', 'PAID', 'AUTOMATIC', 'Contoh tagihan yang sudah lunas.', '2026-09-05 10:19:06.988468', '2026-11-03 12:36:14.243412', 1, NULL),
(2, 'INV-2026-0002', 1, 1, 1, 1, 2, 2, 'Cicilan Oktober', 500000.00, 0.00, 0.00, 500000.00, 200000.00, '2026-10-10', 'PARTIALLY_PAID', 'AUTOMATIC', 'Contoh pembayaran sebagian; sisa Rp300.000.', '2026-09-05 10:19:06.988468', '2026-11-03 12:36:14.243412', 1, NULL),
(3, 'INV-2026-0003', 1, 1, 1, 1, 3, 3, 'Cicilan November', 500000.00, 0.00, 0.00, 500000.00, 0.00, '2026-11-10', 'UNPAID', 'AUTOMATIC', 'Contoh tagihan yang belum dibayar.', '2026-09-05 10:19:06.988468', '2026-09-05 10:19:06.988468', 1, NULL),
(4, 'INV-2026-0004', 2, 1, 1, 2, 16, 1, 'Termin beasiswa sebelum UTS', 2000000.00, 2000000.00, 0.00, 0.00, 0.00, '2026-10-20', 'WAIVED', 'AUTOMATIC', 'Dibebaskan 100% karena beasiswa.', '2026-09-05 10:19:06.988468', '2026-09-05 10:19:06.988468', 1, NULL),
(5, 'INV-2026-0005', 4, 1, 1, 4, 12, 1, 'Pembayaran awal', 400000.00, 0.00, 0.00, 400000.00, 400000.00, '2026-09-20', 'PAID', 'AUTOMATIC', 'Tagihan dari kebijakan khusus marketing.', '2026-09-05 10:19:06.988468', '2026-11-03 12:36:14.243412', 1, NULL),
(6, 'INV-2026-0006', 5, 1, 1, 5, 18, 1, 'Cicilan pertama', 1500000.00, 150000.00, 0.00, 1350000.00, 0.00, '2026-10-20', 'UNPAID', 'AUTOMATIC', 'Contoh potongan khusus Rp150.000.', '2026-09-05 10:19:06.988468', '2026-09-05 10:19:06.988468', 1, NULL),
(7, 'INV-2026-0007', 3, 1, 2, NULL, NULL, NULL, 'Biaya Ujian Susulan', 250000.00, 0.00, 25000.00, 275000.00, 0.00, '2026-08-31', 'OVERDUE', 'MANUAL', 'Tagihan manual dengan denda dan telah melewati jatuh tempo.', '2026-09-05 10:19:06.988468', '2026-11-03 12:36:14.243412', 1, NULL);

-- --------------------------------------------------------

--
-- Table structure for table `student_invoice_adjustments`
--

CREATE TABLE `student_invoice_adjustments` (
  `id` bigint(20) NOT NULL,
  `student_invoice_id` bigint(20) NOT NULL,
  `student_funding_program_id` bigint(20) NOT NULL,
  `adjustment_type` varchar(30) NOT NULL,
  `amount` decimal(18,2) NOT NULL,
  `description` varchar(255) NOT NULL,
  `created_at` datetime(6) NOT NULL
) ;

--
-- Dumping data for table `student_invoice_adjustments`
--

INSERT INTO `student_invoice_adjustments` (`id`, `student_invoice_id`, `student_funding_program_id`, `adjustment_type`, `amount`, `description`, `created_at`) VALUES
(1, 4, 1, 'SCHOLARSHIP', 2000000.00, 'Pembebasan SPP 100% dari Beasiswa Prestasi.', '2026-09-05 10:19:07.156482'),
(2, 6, 2, 'EMPLOYEE_SUBSIDY', 150000.00, 'Potongan Rp150.000 dari program Anak Karyawan.', '2026-09-05 10:19:07.156482');

-- --------------------------------------------------------

--
-- Table structure for table `study_programs`
--

CREATE TABLE `study_programs` (
  `id` bigint(20) NOT NULL,
  `code` varchar(20) NOT NULL,
  `name` varchar(150) NOT NULL,
  `education_level` varchar(20) NOT NULL,
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `created_at` datetime(6) NOT NULL,
  `updated_at` datetime(6) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Dumping data for table `study_programs`
--

INSERT INTO `study_programs` (`id`, `code`, `name`, `education_level`, `is_active`, `created_at`, `updated_at`) VALUES
(1, 'TI', 'Teknik Informatika', 'S1', 1, '2026-09-05 10:19:06.571956', '2026-09-05 10:19:06.571956'),
(2, 'SI', 'Sistem Informasi', 'S1', 1, '2026-09-05 10:19:06.571956', '2026-09-05 10:19:06.571956');

-- --------------------------------------------------------

--
-- Table structure for table `users`
--

CREATE TABLE `users` (
  `Id` varchar(255) NOT NULL,
  `UserName` varchar(256) DEFAULT NULL,
  `NormalizedUserName` varchar(256) DEFAULT NULL,
  `Email` varchar(256) DEFAULT NULL,
  `NormalizedEmail` varchar(256) DEFAULT NULL,
  `EmailConfirmed` tinyint(1) NOT NULL,
  `SecurityStamp` longtext DEFAULT NULL,
  `PhoneNumber` longtext DEFAULT NULL,
  `PhoneNumberConfirmed` tinyint(1) NOT NULL,
  `TwoFactorEnabled` tinyint(1) NOT NULL,
  `LockoutEnd` datetime DEFAULT NULL,
  `LockoutEnabled` tinyint(1) NOT NULL,
  `AccessFailedCount` int(11) NOT NULL,
  `GoPasswordHash` longtext DEFAULT NULL,
  `GoPasswordResetRequired` tinyint(1) NOT NULL DEFAULT 1,
  `GoPasswordUpdatedAt` datetime(6) DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Dumping data for table `users`
--

INSERT INTO `users` (`Id`, `UserName`, `NormalizedUserName`, `Email`, `NormalizedEmail`, `EmailConfirmed`, `SecurityStamp`, `PhoneNumber`, `PhoneNumberConfirmed`, `TwoFactorEnabled`, `LockoutEnd`, `LockoutEnabled`, `AccessFailedCount`, `GoPasswordHash`, `GoPasswordResetRequired`, `GoPasswordUpdatedAt`) VALUES
('234c6bce-100d-49ef-b9f9-64ec04377ad4', 'marketing@siato.local', 'MARKETING@SIATO.LOCAL', 'marketing@siato.local', 'MARKETING@SIATO.LOCAL', 1, '8h9c6dqABMTVRut0k0WPoRkqlbClx5xNT0REifj5IuQ', NULL, 0, 0, NULL, 1, 0, '$argon2id$v=19$m=65536,t=3,p=2$OXORQhA8CaMnGcs6wbQ51w$WdTEbXdObKQlBjgthpolAEHxpnKAOhJB2u70MGt8GoA', 0, '2026-11-03 13:31:38.788828'),
('54434c08-936f-4e54-ab0b-565319c28f6f', 'superadmin@siato.local', 'SUPERADMIN@SIATO.LOCAL', 'superadmin@siato.local', 'SUPERADMIN@SIATO.LOCAL', 1, 'RMBM6GmQIFqy0dVy1ISOjHuMZEdkceCeaRKrgj9WOlk', NULL, 0, 0, NULL, 1, 0, '$argon2id$v=19$m=65536,t=3,p=2$xwCN6EiNtM4TXs2GD3mrjw$1E0eYv/2u7X8c8+AOd3+vptB+2Xa+wk0YGzMwUz1lI8', 0, '2026-11-03 13:31:38.488076'),
('69c1461b-12f7-4814-ac9d-e059915486a1', 'rektor@siato.local', 'REKTOR@SIATO.LOCAL', 'rektor@siato.local', 'REKTOR@SIATO.LOCAL', 1, 'xLKGQuHOwOaByy8trWWJupZ25ioA6aETnbzgZNa1Tpc', NULL, 0, 0, NULL, 1, 0, '$argon2id$v=19$m=65536,t=3,p=2$zE8kOANvwGU4Gubz8JXrjQ$fyLfvIAyoWPoEkfsNRKbTtQKZJEviHL+xNWAv+kU8Sk', 0, '2026-11-03 13:31:38.665312'),
('b713ec29-747b-40d5-bb98-261110e190d1', 'akademik@siato.local', 'AKADEMIK@SIATO.LOCAL', 'akademik@siato.local', 'AKADEMIK@SIATO.LOCAL', 1, 'dwloT2F3mq1OM0RDfE_OramtygdueqTB75vQDT2k-so', NULL, 0, 0, NULL, 1, 0, '$argon2id$v=19$m=65536,t=3,p=2$FEn3EkgQ1dp2Nu/YFUeckA$I2jixPC3hY5Xj9BpRZfl4ANENOy9PTsi5Ml2AbncR/g', 0, '2026-11-03 13:31:38.727336'),
('b7166450-9277-4f2e-ace1-37dddeae02a6', 'manager@siato.local', 'MANAGER@SIATO.LOCAL', 'manager@siato.local', 'MANAGER@SIATO.LOCAL', 1, 'xrRxX342HA3i05lNBpv7WXIIU-5ZXpwnwJM1kfdOyYQ', NULL, 0, 0, NULL, 1, 0, '$argon2id$v=19$m=65536,t=3,p=2$/zNYHZEOrXtbW4io10qaJQ$x/RNfQ9ceYCr2c1aUPHdWfpSlM5QUFFUFhiKdcwLSco', 0, '2026-11-03 13:31:38.601595'),
('e5010e83-353e-4d06-8589-fd9245750e39', 'keuangan@siato.local', 'KEUANGAN@SIATO.LOCAL', 'keuangan@siato.local', 'KEUANGAN@SIATO.LOCAL', 1, '5Wu9_5QCwuTR4S6_jUw47b9KvGLF4ZpEX3pB8-Sbusc', NULL, 0, 0, NULL, 1, 0, '$argon2id$v=19$m=65536,t=3,p=2$O4BJh3eXVA/K7N2hFBeQHw$YNHSNJUCa2xwhtSJujJJBPbdT60C6P++tu2NPZwgdJA', 0, '2026-11-03 13:31:38.544750');

-- --------------------------------------------------------

--
-- Table structure for table `user_claims`
--

CREATE TABLE `user_claims` (
  `Id` int(11) NOT NULL,
  `UserId` varchar(255) NOT NULL,
  `ClaimType` longtext DEFAULT NULL,
  `ClaimValue` longtext DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- Table structure for table `user_logins`
--

CREATE TABLE `user_logins` (
  `LoginProvider` varchar(255) NOT NULL,
  `ProviderKey` varchar(255) NOT NULL,
  `ProviderDisplayName` longtext DEFAULT NULL,
  `UserId` varchar(255) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- Table structure for table `user_permissions`
--

CREATE TABLE `user_permissions` (
  `user_id` varchar(255) NOT NULL,
  `permission_id` bigint(20) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- Table structure for table `user_roles`
--

CREATE TABLE `user_roles` (
  `UserId` varchar(255) NOT NULL,
  `RoleId` varchar(255) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Dumping data for table `user_roles`
--

INSERT INTO `user_roles` (`UserId`, `RoleId`) VALUES
('234c6bce-100d-49ef-b9f9-64ec04377ad4', '356113ab-ea7b-45b8-b082-0f3882aec232'),
('54434c08-936f-4e54-ab0b-565319c28f6f', 'bdfd09a6-f570-4ae8-97f9-00f9723cc123'),
('69c1461b-12f7-4814-ac9d-e059915486a1', '915976e6-3b8a-4866-8f81-d3d4dd533ba2'),
('b713ec29-747b-40d5-bb98-261110e190d1', '341c5cd7-ce65-4df7-b4a7-b9ae302558cd'),
('b7166450-9277-4f2e-ace1-37dddeae02a6', 'f04f98fb-db01-41b5-9865-c020f16e7ae4'),
('e5010e83-353e-4d06-8589-fd9245750e39', '50c7e970-09cc-44b2-9ef7-3eb9566620d6');

-- --------------------------------------------------------

--
-- Table structure for table `user_tokens`
--

CREATE TABLE `user_tokens` (
  `UserId` varchar(255) NOT NULL,
  `LoginProvider` varchar(255) NOT NULL,
  `Name` varchar(255) NOT NULL,
  `Value` longtext DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- Table structure for table `__efmigrationshistory`
--

CREATE TABLE `__efmigrationshistory` (
  `MigrationId` varchar(150) NOT NULL,
  `ProductVersion` varchar(32) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Dumping data for table `__efmigrationshistory`
--

INSERT INTO `__efmigrationshistory` (`MigrationId`, `ProductVersion`) VALUES
('20260902070834_InitialIdentityPermissions', '10.0.11'),
('20260904032510_AddAcademicMasterTables', '10.0.11'),
('20260904060103_AddStudentCategoryToStudents', '10.0.11'),
('20260904063410_AddStudentCohortYearClass', '10.0.11'),
('20260904064552_AddBillingPolicies', '10.0.11'),
('20260904072445_AddStudentBillingPolicies', '10.0.11'),
('20260905010535_AddStudentInvoices', '10.0.11'),
('20260905020308_AddPaymentsAndAllocations', '10.0.11'),
('20260905031300_AddFundingProgramsAndPayers', '10.0.11'),
('20260905044154_AddFundingBenefitApplicationRules', '10.0.11'),
('20260905121356_AddFinancialAuditAndConcurrency', '10.0.11'),
('20260905125934_AddInvoiceGenerationBatches', '10.0.11'),
('20260905134140_AddStudentBillingPolicyBatches', '10.0.11'),
('20260906000000_AddPaymentProof', '10.0.11'),
('20260906000001_EnsurePaymentProofColumn', '10.0.11');

--
-- Indexes for dumped tables
--

--
-- Indexes for table `academic_periods`
--
ALTER TABLE `academic_periods`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `IX_academic_periods_academic_year_semester` (`academic_year`,`semester`),
  ADD UNIQUE KEY `IX_academic_periods_code` (`code`);

--
-- Indexes for table `audit_logs`
--
ALTER TABLE `audit_logs`
  ADD PRIMARY KEY (`id`),
  ADD KEY `IX_audit_logs_entity_type_entity_id_performed_at` (`entity_type`,`entity_id`,`performed_at`),
  ADD KEY `IX_audit_logs_performed_by_id` (`performed_by_id`);

--
-- Indexes for table `billing_policies`
--
ALTER TABLE `billing_policies`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `IX_billing_policies_code` (`code`),
  ADD KEY `IX_billing_policies_academic_period_id` (`academic_period_id`),
  ADD KEY `IX_billing_policies_charge_type_id` (`charge_type_id`),
  ADD KEY `IX_billing_policies_cohort_year_cohort_year_class` (`cohort_year`,`cohort_year_class`),
  ADD KEY `IX_billing_policies_student_category_id` (`student_category_id`),
  ADD KEY `IX_billing_policies_study_program_id` (`study_program_id`);

--
-- Indexes for table `billing_policy_schedules`
--
ALTER TABLE `billing_policy_schedules`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `IX_billing_policy_schedules_billing_policy_id_sequence_no` (`billing_policy_id`,`sequence_no`),
  ADD KEY `IX_billing_policy_schedules_due_date` (`due_date`);

--
-- Indexes for table `charge_types`
--
ALTER TABLE `charge_types`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `IX_charge_types_code` (`code`);

--
-- Indexes for table `funding_programs`
--
ALTER TABLE `funding_programs`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `IX_funding_programs_code` (`code`),
  ADD KEY `IX_funding_programs_program_type_is_active` (`program_type`,`is_active`);

--
-- Indexes for table `funding_program_benefits`
--
ALTER TABLE `funding_program_benefits`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `IX_funding_program_benefits_funding_program_id_charge_type_id` (`funding_program_id`,`charge_type_id`),
  ADD KEY `IX_funding_program_benefits_charge_type_id` (`charge_type_id`);

--
-- Indexes for table `go_auth_sessions`
--
ALTER TABLE `go_auth_sessions`
  ADD PRIMARY KEY (`token_hash`),
  ADD KEY `IX_go_auth_sessions_user_id` (`user_id`),
  ADD KEY `IX_go_auth_sessions_expires_at` (`expires_at`);

--
-- Indexes for table `go_password_reset_tokens`
--
ALTER TABLE `go_password_reset_tokens`
  ADD PRIMARY KEY (`token_hash`),
  ADD KEY `IX_go_password_reset_tokens_user_id` (`user_id`),
  ADD KEY `IX_go_password_reset_tokens_expires_at` (`expires_at`);

--
-- Indexes for table `invoice_generation_batches`
--
ALTER TABLE `invoice_generation_batches`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `IX_invoice_generation_batches_batch_number` (`batch_number`),
  ADD KEY `IX_invoice_generation_batches_academic_period_id` (`academic_period_id`),
  ADD KEY `IX_invoice_generation_batches_cancelled_by_id` (`cancelled_by_id`),
  ADD KEY `IX_invoice_generation_batches_initiated_by_id` (`initiated_by_id`),
  ADD KEY `IX_invoice_generation_batches_status_started_at` (`status`,`started_at`);

--
-- Indexes for table `invoice_generation_batch_items`
--
ALTER TABLE `invoice_generation_batch_items`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `IX_invoice_generation_batch_items_invoice_generation_batch_id_s~` (`invoice_generation_batch_id`,`student_billing_policy_id`,`billing_policy_schedule_id`),
  ADD KEY `IX_invoice_generation_batch_items_billing_policy_schedule_id` (`billing_policy_schedule_id`),
  ADD KEY `IX_invoice_generation_batch_items_student_billing_policy_id` (`student_billing_policy_id`),
  ADD KEY `IX_invoice_generation_batch_items_student_invoice_id` (`student_invoice_id`);

--
-- Indexes for table `payments`
--
ALTER TABLE `payments`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `IX_payments_payment_number` (`payment_number`),
  ADD UNIQUE KEY `IX_payments_reference_number` (`reference_number`),
  ADD KEY `IX_payments_received_by_id` (`received_by_id`),
  ADD KEY `IX_payments_status_payment_date` (`status`,`payment_date`),
  ADD KEY `IX_payments_student_id_payment_date` (`student_id`,`payment_date`),
  ADD KEY `IX_payments_verified_by_id` (`verified_by_id`),
  ADD KEY `IX_payments_funding_program_id` (`funding_program_id`),
  ADD KEY `IX_payments_cancelled_by_id` (`cancelled_by_id`);

--
-- Indexes for table `payment_allocations`
--
ALTER TABLE `payment_allocations`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `IX_payment_allocations_payment_id_student_invoice_id` (`payment_id`,`student_invoice_id`),
  ADD KEY `IX_payment_allocations_student_invoice_id` (`student_invoice_id`);

--
-- Indexes for table `permissions`
--
ALTER TABLE `permissions`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `IX_permissions_name_guard_name` (`name`,`guard_name`);

--
-- Indexes for table `roles`
--
ALTER TABLE `roles`
  ADD PRIMARY KEY (`Id`),
  ADD UNIQUE KEY `RoleNameIndex` (`NormalizedName`);

--
-- Indexes for table `role_claims`
--
ALTER TABLE `role_claims`
  ADD PRIMARY KEY (`Id`),
  ADD KEY `IX_role_claims_RoleId` (`RoleId`);

--
-- Indexes for table `role_permissions`
--
ALTER TABLE `role_permissions`
  ADD PRIMARY KEY (`role_id`,`permission_id`),
  ADD KEY `IX_role_permissions_permission_id` (`permission_id`);

--
-- Indexes for table `students`
--
ALTER TABLE `students`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `IX_students_nim` (`nim`),
  ADD KEY `IX_students_cohort_year_status` (`cohort_year`,`status`),
  ADD KEY `IX_students_study_program_id` (`study_program_id`),
  ADD KEY `IX_students_student_category_id` (`student_category_id`),
  ADD KEY `IX_students_cohort_year_class` (`cohort_year_class`);

--
-- Indexes for table `student_billing_policies`
--
ALTER TABLE `student_billing_policies`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `IX_student_billing_policies_student_id_billing_policy_id` (`student_id`,`billing_policy_id`),
  ADD KEY `IX_student_billing_policies_approved_by_id` (`approved_by_id`),
  ADD KEY `IX_student_billing_policies_billing_policy_id` (`billing_policy_id`),
  ADD KEY `IX_student_billing_policies_status_assigned_at` (`status`,`assigned_at`);

--
-- Indexes for table `student_billing_policy_batches`
--
ALTER TABLE `student_billing_policy_batches`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `IX_student_billing_policy_batches_batch_number` (`batch_number`),
  ADD KEY `IX_student_billing_policy_batches_billing_policy_id` (`billing_policy_id`),
  ADD KEY `IX_student_billing_policy_batches_cancelled_by_id` (`cancelled_by_id`),
  ADD KEY `IX_student_billing_policy_batches_initiated_by_id` (`initiated_by_id`),
  ADD KEY `IX_student_billing_policy_batches_student_category_id` (`student_category_id`),
  ADD KEY `IX_student_billing_policy_batches_study_program_id` (`study_program_id`);

--
-- Indexes for table `student_billing_policy_batch_items`
--
ALTER TABLE `student_billing_policy_batch_items`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `IX_student_billing_policy_batch_items_student_billing_policy_ba~` (`student_billing_policy_batch_id`,`student_id`),
  ADD KEY `IX_student_billing_policy_batch_items_student_billing_policy_id` (`student_billing_policy_id`),
  ADD KEY `IX_student_billing_policy_batch_items_student_id` (`student_id`);

--
-- Indexes for table `student_categories`
--
ALTER TABLE `student_categories`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `IX_student_categories_code` (`code`);

--
-- Indexes for table `student_funding_programs`
--
ALTER TABLE `student_funding_programs`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `IX_student_funding_programs_student_id_funding_program_id_acade~` (`student_id`,`funding_program_id`,`academic_period_id`),
  ADD KEY `IX_student_funding_programs_academic_period_id` (`academic_period_id`),
  ADD KEY `IX_student_funding_programs_funding_program_id` (`funding_program_id`);

--
-- Indexes for table `student_invoices`
--
ALTER TABLE `student_invoices`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `IX_student_invoices_invoice_number` (`invoice_number`),
  ADD UNIQUE KEY `IX_student_invoices_student_id_billing_policy_schedule_id` (`student_id`,`billing_policy_schedule_id`),
  ADD KEY `IX_student_invoices_academic_period_id` (`academic_period_id`),
  ADD KEY `IX_student_invoices_billing_policy_schedule_id` (`billing_policy_schedule_id`),
  ADD KEY `IX_student_invoices_charge_type_id` (`charge_type_id`),
  ADD KEY `IX_student_invoices_student_billing_policy_id` (`student_billing_policy_id`),
  ADD KEY `IX_student_invoices_student_id_status_due_date` (`student_id`,`status`,`due_date`),
  ADD KEY `IX_student_invoices_invoice_generation_batch_id` (`invoice_generation_batch_id`);

--
-- Indexes for table `student_invoice_adjustments`
--
ALTER TABLE `student_invoice_adjustments`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `IX_student_invoice_adjustments_student_invoice_id_student_fundi~` (`student_invoice_id`,`student_funding_program_id`),
  ADD KEY `IX_student_invoice_adjustments_student_funding_program_id` (`student_funding_program_id`);

--
-- Indexes for table `study_programs`
--
ALTER TABLE `study_programs`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `IX_study_programs_code` (`code`);

--
-- Indexes for table `users`
--
ALTER TABLE `users`
  ADD PRIMARY KEY (`Id`),
  ADD UNIQUE KEY `UserNameIndex` (`NormalizedUserName`),
  ADD KEY `EmailIndex` (`NormalizedEmail`);

--
-- Indexes for table `user_claims`
--
ALTER TABLE `user_claims`
  ADD PRIMARY KEY (`Id`),
  ADD KEY `IX_user_claims_UserId` (`UserId`);

--
-- Indexes for table `user_logins`
--
ALTER TABLE `user_logins`
  ADD PRIMARY KEY (`LoginProvider`,`ProviderKey`),
  ADD KEY `IX_user_logins_UserId` (`UserId`);

--
-- Indexes for table `user_permissions`
--
ALTER TABLE `user_permissions`
  ADD PRIMARY KEY (`user_id`,`permission_id`),
  ADD KEY `IX_user_permissions_permission_id` (`permission_id`);

--
-- Indexes for table `user_roles`
--
ALTER TABLE `user_roles`
  ADD PRIMARY KEY (`UserId`,`RoleId`),
  ADD KEY `IX_user_roles_RoleId` (`RoleId`);

--
-- Indexes for table `user_tokens`
--
ALTER TABLE `user_tokens`
  ADD PRIMARY KEY (`UserId`,`LoginProvider`,`Name`);

--
-- Indexes for table `__efmigrationshistory`
--
ALTER TABLE `__efmigrationshistory`
  ADD PRIMARY KEY (`MigrationId`);

--
-- AUTO_INCREMENT for dumped tables
--

--
-- AUTO_INCREMENT for table `academic_periods`
--
ALTER TABLE `academic_periods`
  MODIFY `id` bigint(20) NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `audit_logs`
--
ALTER TABLE `audit_logs`
  MODIFY `id` bigint(20) NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `billing_policies`
--
ALTER TABLE `billing_policies`
  MODIFY `id` bigint(20) NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `billing_policy_schedules`
--
ALTER TABLE `billing_policy_schedules`
  MODIFY `id` bigint(20) NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `charge_types`
--
ALTER TABLE `charge_types`
  MODIFY `id` bigint(20) NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=3;

--
-- AUTO_INCREMENT for table `funding_programs`
--
ALTER TABLE `funding_programs`
  MODIFY `id` bigint(20) NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `funding_program_benefits`
--
ALTER TABLE `funding_program_benefits`
  MODIFY `id` bigint(20) NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `invoice_generation_batches`
--
ALTER TABLE `invoice_generation_batches`
  MODIFY `id` bigint(20) NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `invoice_generation_batch_items`
--
ALTER TABLE `invoice_generation_batch_items`
  MODIFY `id` bigint(20) NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `payments`
--
ALTER TABLE `payments`
  MODIFY `id` bigint(20) NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `payment_allocations`
--
ALTER TABLE `payment_allocations`
  MODIFY `id` bigint(20) NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `permissions`
--
ALTER TABLE `permissions`
  MODIFY `id` bigint(20) NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=106;

--
-- AUTO_INCREMENT for table `role_claims`
--
ALTER TABLE `role_claims`
  MODIFY `Id` int(11) NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `students`
--
ALTER TABLE `students`
  MODIFY `id` bigint(20) NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=6;

--
-- AUTO_INCREMENT for table `student_billing_policies`
--
ALTER TABLE `student_billing_policies`
  MODIFY `id` bigint(20) NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `student_billing_policy_batches`
--
ALTER TABLE `student_billing_policy_batches`
  MODIFY `id` bigint(20) NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `student_billing_policy_batch_items`
--
ALTER TABLE `student_billing_policy_batch_items`
  MODIFY `id` bigint(20) NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `student_categories`
--
ALTER TABLE `student_categories`
  MODIFY `id` bigint(20) NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=5;

--
-- AUTO_INCREMENT for table `student_funding_programs`
--
ALTER TABLE `student_funding_programs`
  MODIFY `id` bigint(20) NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `student_invoices`
--
ALTER TABLE `student_invoices`
  MODIFY `id` bigint(20) NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `student_invoice_adjustments`
--
ALTER TABLE `student_invoice_adjustments`
  MODIFY `id` bigint(20) NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `study_programs`
--
ALTER TABLE `study_programs`
  MODIFY `id` bigint(20) NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=3;

--
-- AUTO_INCREMENT for table `user_claims`
--
ALTER TABLE `user_claims`
  MODIFY `Id` int(11) NOT NULL AUTO_INCREMENT;

--
-- Constraints for dumped tables
--

--
-- Constraints for table `audit_logs`
--
ALTER TABLE `audit_logs`
  ADD CONSTRAINT `FK_audit_logs_users_performed_by_id` FOREIGN KEY (`performed_by_id`) REFERENCES `users` (`Id`) ON DELETE SET NULL;

--
-- Constraints for table `billing_policies`
--
ALTER TABLE `billing_policies`
  ADD CONSTRAINT `FK_billing_policies_academic_periods_academic_period_id` FOREIGN KEY (`academic_period_id`) REFERENCES `academic_periods` (`id`),
  ADD CONSTRAINT `FK_billing_policies_charge_types_charge_type_id` FOREIGN KEY (`charge_type_id`) REFERENCES `charge_types` (`id`),
  ADD CONSTRAINT `FK_billing_policies_student_categories_student_category_id` FOREIGN KEY (`student_category_id`) REFERENCES `student_categories` (`id`),
  ADD CONSTRAINT `FK_billing_policies_study_programs_study_program_id` FOREIGN KEY (`study_program_id`) REFERENCES `study_programs` (`id`);

--
-- Constraints for table `billing_policy_schedules`
--
ALTER TABLE `billing_policy_schedules`
  ADD CONSTRAINT `FK_billing_policy_schedules_billing_policies_billing_policy_id` FOREIGN KEY (`billing_policy_id`) REFERENCES `billing_policies` (`id`) ON DELETE CASCADE;

--
-- Constraints for table `funding_program_benefits`
--
ALTER TABLE `funding_program_benefits`
  ADD CONSTRAINT `FK_funding_program_benefits_charge_types_charge_type_id` FOREIGN KEY (`charge_type_id`) REFERENCES `charge_types` (`id`),
  ADD CONSTRAINT `FK_funding_program_benefits_funding_programs_funding_program_id` FOREIGN KEY (`funding_program_id`) REFERENCES `funding_programs` (`id`) ON DELETE CASCADE;

--
-- Constraints for table `go_auth_sessions`
--
ALTER TABLE `go_auth_sessions`
  ADD CONSTRAINT `FK_go_auth_sessions_users_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`Id`) ON DELETE CASCADE;

--
-- Constraints for table `go_password_reset_tokens`
--
ALTER TABLE `go_password_reset_tokens`
  ADD CONSTRAINT `FK_go_password_reset_tokens_users_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`Id`) ON DELETE CASCADE;

--
-- Constraints for table `invoice_generation_batches`
--
ALTER TABLE `invoice_generation_batches`
  ADD CONSTRAINT `FK_invoice_generation_batches_academic_periods_academic_period_~` FOREIGN KEY (`academic_period_id`) REFERENCES `academic_periods` (`id`),
  ADD CONSTRAINT `FK_invoice_generation_batches_users_cancelled_by_id` FOREIGN KEY (`cancelled_by_id`) REFERENCES `users` (`Id`) ON DELETE SET NULL,
  ADD CONSTRAINT `FK_invoice_generation_batches_users_initiated_by_id` FOREIGN KEY (`initiated_by_id`) REFERENCES `users` (`Id`) ON DELETE SET NULL;

--
-- Constraints for table `invoice_generation_batch_items`
--
ALTER TABLE `invoice_generation_batch_items`
  ADD CONSTRAINT `FK_invoice_generation_batch_items_billing_policy_schedules_bill~` FOREIGN KEY (`billing_policy_schedule_id`) REFERENCES `billing_policy_schedules` (`id`),
  ADD CONSTRAINT `FK_invoice_generation_batch_items_invoice_generation_batches_in~` FOREIGN KEY (`invoice_generation_batch_id`) REFERENCES `invoice_generation_batches` (`id`) ON DELETE CASCADE,
  ADD CONSTRAINT `FK_invoice_generation_batch_items_student_billing_policies_stud~` FOREIGN KEY (`student_billing_policy_id`) REFERENCES `student_billing_policies` (`id`),
  ADD CONSTRAINT `FK_invoice_generation_batch_items_student_invoices_student_invo~` FOREIGN KEY (`student_invoice_id`) REFERENCES `student_invoices` (`id`) ON DELETE SET NULL;

--
-- Constraints for table `payments`
--
ALTER TABLE `payments`
  ADD CONSTRAINT `FK_payments_funding_programs_funding_program_id` FOREIGN KEY (`funding_program_id`) REFERENCES `funding_programs` (`id`),
  ADD CONSTRAINT `FK_payments_students_student_id` FOREIGN KEY (`student_id`) REFERENCES `students` (`id`),
  ADD CONSTRAINT `FK_payments_users_cancelled_by_id` FOREIGN KEY (`cancelled_by_id`) REFERENCES `users` (`Id`) ON DELETE SET NULL,
  ADD CONSTRAINT `FK_payments_users_received_by_id` FOREIGN KEY (`received_by_id`) REFERENCES `users` (`Id`),
  ADD CONSTRAINT `FK_payments_users_verified_by_id` FOREIGN KEY (`verified_by_id`) REFERENCES `users` (`Id`);

--
-- Constraints for table `payment_allocations`
--
ALTER TABLE `payment_allocations`
  ADD CONSTRAINT `FK_payment_allocations_payments_payment_id` FOREIGN KEY (`payment_id`) REFERENCES `payments` (`id`),
  ADD CONSTRAINT `FK_payment_allocations_student_invoices_student_invoice_id` FOREIGN KEY (`student_invoice_id`) REFERENCES `student_invoices` (`id`);

--
-- Constraints for table `role_claims`
--
ALTER TABLE `role_claims`
  ADD CONSTRAINT `FK_role_claims_roles_RoleId` FOREIGN KEY (`RoleId`) REFERENCES `roles` (`Id`) ON DELETE CASCADE;

--
-- Constraints for table `role_permissions`
--
ALTER TABLE `role_permissions`
  ADD CONSTRAINT `FK_role_permissions_permissions_permission_id` FOREIGN KEY (`permission_id`) REFERENCES `permissions` (`id`) ON DELETE CASCADE,
  ADD CONSTRAINT `FK_role_permissions_roles_role_id` FOREIGN KEY (`role_id`) REFERENCES `roles` (`Id`) ON DELETE CASCADE;

--
-- Constraints for table `students`
--
ALTER TABLE `students`
  ADD CONSTRAINT `FK_students_student_categories_student_category_id` FOREIGN KEY (`student_category_id`) REFERENCES `student_categories` (`id`),
  ADD CONSTRAINT `FK_students_study_programs_study_program_id` FOREIGN KEY (`study_program_id`) REFERENCES `study_programs` (`id`);

--
-- Constraints for table `student_billing_policies`
--
ALTER TABLE `student_billing_policies`
  ADD CONSTRAINT `FK_student_billing_policies_billing_policies_billing_policy_id` FOREIGN KEY (`billing_policy_id`) REFERENCES `billing_policies` (`id`),
  ADD CONSTRAINT `FK_student_billing_policies_students_student_id` FOREIGN KEY (`student_id`) REFERENCES `students` (`id`),
  ADD CONSTRAINT `FK_student_billing_policies_users_approved_by_id` FOREIGN KEY (`approved_by_id`) REFERENCES `users` (`Id`) ON DELETE SET NULL;

--
-- Constraints for table `student_billing_policy_batches`
--
ALTER TABLE `student_billing_policy_batches`
  ADD CONSTRAINT `FK_student_billing_policy_batches_billing_policies_billing_poli~` FOREIGN KEY (`billing_policy_id`) REFERENCES `billing_policies` (`id`),
  ADD CONSTRAINT `FK_student_billing_policy_batches_student_categories_student_ca~` FOREIGN KEY (`student_category_id`) REFERENCES `student_categories` (`id`),
  ADD CONSTRAINT `FK_student_billing_policy_batches_study_programs_study_program_~` FOREIGN KEY (`study_program_id`) REFERENCES `study_programs` (`id`),
  ADD CONSTRAINT `FK_student_billing_policy_batches_users_cancelled_by_id` FOREIGN KEY (`cancelled_by_id`) REFERENCES `users` (`Id`) ON DELETE SET NULL,
  ADD CONSTRAINT `FK_student_billing_policy_batches_users_initiated_by_id` FOREIGN KEY (`initiated_by_id`) REFERENCES `users` (`Id`) ON DELETE SET NULL;

--
-- Constraints for table `student_billing_policy_batch_items`
--
ALTER TABLE `student_billing_policy_batch_items`
  ADD CONSTRAINT `FK_student_billing_policy_batch_items_student_billing_policies_~` FOREIGN KEY (`student_billing_policy_id`) REFERENCES `student_billing_policies` (`id`) ON DELETE SET NULL,
  ADD CONSTRAINT `FK_student_billing_policy_batch_items_student_billing_policy_ba~` FOREIGN KEY (`student_billing_policy_batch_id`) REFERENCES `student_billing_policy_batches` (`id`) ON DELETE CASCADE,
  ADD CONSTRAINT `FK_student_billing_policy_batch_items_students_student_id` FOREIGN KEY (`student_id`) REFERENCES `students` (`id`);

--
-- Constraints for table `student_funding_programs`
--
ALTER TABLE `student_funding_programs`
  ADD CONSTRAINT `FK_student_funding_programs_academic_periods_academic_period_id` FOREIGN KEY (`academic_period_id`) REFERENCES `academic_periods` (`id`),
  ADD CONSTRAINT `FK_student_funding_programs_funding_programs_funding_program_id` FOREIGN KEY (`funding_program_id`) REFERENCES `funding_programs` (`id`),
  ADD CONSTRAINT `FK_student_funding_programs_students_student_id` FOREIGN KEY (`student_id`) REFERENCES `students` (`id`);

--
-- Constraints for table `student_invoices`
--
ALTER TABLE `student_invoices`
  ADD CONSTRAINT `FK_student_invoices_academic_periods_academic_period_id` FOREIGN KEY (`academic_period_id`) REFERENCES `academic_periods` (`id`),
  ADD CONSTRAINT `FK_student_invoices_billing_policy_schedules_billing_policy_sch~` FOREIGN KEY (`billing_policy_schedule_id`) REFERENCES `billing_policy_schedules` (`id`),
  ADD CONSTRAINT `FK_student_invoices_charge_types_charge_type_id` FOREIGN KEY (`charge_type_id`) REFERENCES `charge_types` (`id`),
  ADD CONSTRAINT `FK_student_invoices_invoice_generation_batches_invoice_generati~` FOREIGN KEY (`invoice_generation_batch_id`) REFERENCES `invoice_generation_batches` (`id`),
  ADD CONSTRAINT `FK_student_invoices_student_billing_policies_student_billing_po~` FOREIGN KEY (`student_billing_policy_id`) REFERENCES `student_billing_policies` (`id`),
  ADD CONSTRAINT `FK_student_invoices_students_student_id` FOREIGN KEY (`student_id`) REFERENCES `students` (`id`);

--
-- Constraints for table `student_invoice_adjustments`
--
ALTER TABLE `student_invoice_adjustments`
  ADD CONSTRAINT `FK_student_invoice_adjustments_student_funding_programs_student~` FOREIGN KEY (`student_funding_program_id`) REFERENCES `student_funding_programs` (`id`),
  ADD CONSTRAINT `FK_student_invoice_adjustments_student_invoices_student_invoice~` FOREIGN KEY (`student_invoice_id`) REFERENCES `student_invoices` (`id`);

--
-- Constraints for table `user_claims`
--
ALTER TABLE `user_claims`
  ADD CONSTRAINT `FK_user_claims_users_UserId` FOREIGN KEY (`UserId`) REFERENCES `users` (`Id`) ON DELETE CASCADE;

--
-- Constraints for table `user_logins`
--
ALTER TABLE `user_logins`
  ADD CONSTRAINT `FK_user_logins_users_UserId` FOREIGN KEY (`UserId`) REFERENCES `users` (`Id`) ON DELETE CASCADE;

--
-- Constraints for table `user_permissions`
--
ALTER TABLE `user_permissions`
  ADD CONSTRAINT `FK_user_permissions_permissions_permission_id` FOREIGN KEY (`permission_id`) REFERENCES `permissions` (`id`) ON DELETE CASCADE,
  ADD CONSTRAINT `FK_user_permissions_users_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`Id`) ON DELETE CASCADE;

--
-- Constraints for table `user_roles`
--
ALTER TABLE `user_roles`
  ADD CONSTRAINT `FK_user_roles_roles_RoleId` FOREIGN KEY (`RoleId`) REFERENCES `roles` (`Id`) ON DELETE CASCADE,
  ADD CONSTRAINT `FK_user_roles_users_UserId` FOREIGN KEY (`UserId`) REFERENCES `users` (`Id`) ON DELETE CASCADE;

--
-- Constraints for table `user_tokens`
--
ALTER TABLE `user_tokens`
  ADD CONSTRAINT `FK_user_tokens_users_UserId` FOREIGN KEY (`UserId`) REFERENCES `users` (`Id`) ON DELETE CASCADE;
COMMIT;

/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
