Saya mohon maaf atas ketidaknyamanan ini. Sepertinya ada kesalahan teknis pada respons sebelumnya yang menyebabkan munculnya teks **``** dan format sitasi yang kurang rapi di dalam daftar (*bullet points*).

Ini adalah bagian dari draf *README* yang saya berikan sebelumnya, dan seharusnya terlihat bersih dan terstruktur.

Berikut adalah draf *README* yang sudah diperbaiki, bersih dari kode sitasi yang mengganggu, dan siap Anda gunakan di GitHub:

---

## README Proyek: Sistem Pelaporan Prestasi Mahasiswa

### Gambaran Umum Proyek

**Sistem Pelaporan Prestasi Mahasiswa** adalah aplikasi *back-end* berbasis **REST API** yang dirancang untuk mengelola proses pelaporan, verifikasi, dan pengelolaan data prestasi akademik dan non-akademik mahasiswa. Sistem ini mendukung arsitektur *multi-role* dan menyediakan fleksibilitas data dengan dukungan *field* prestasi yang dinamis.

---

### Fitur Utama (Functionalitas)

* **Autentikasi & Otorisasi:** Pengguna dapat *login* menggunakan *username/email* dan *password*. Setiap *endpoint* dilindungi dengan mekanisme **Role-Based Access Control (RBAC)** menggunakan **JSON Web Token (JWT)**.
* **Manajemen Prestasi Mahasiswa:** Mahasiswa dapat menambahkan, *read*, dan *update* prestasi mereka sendiri.
* **Workflow Prestasi:** Mahasiswa dapat mengubah status prestasi dari **'draft'** menjadi **'submitted'** untuk diverifikasi.
* **Verifikasi Prestasi (Dosen Wali):** Dosen Wali dapat melihat daftar prestasi mahasiswa bimbingannya. Mereka dapat memverifikasi (**'verified'**) atau menolak (**'rejected'**) prestasi yang berstatus **'submitted'**.
* **Manajemen Pengguna (Admin):** Admin memiliki *full access* untuk *CRUD users* dan menetapkan *roles*.
* **Pelaporan dan Analitik:** Mampu menghasilkan statistik prestasi, seperti total per tipe, per periode, top mahasiswa berprestasi, dan distribusi tingkat kompetisi.

---

### Teknologi yang Digunakan

| Kategori | Teknologi | Deskripsi |
| :--- | :--- | :--- |
| **Arsitektur** | REST API | Aplikasi *back-end* berbasis Representational State Transfer API. |
| **Database Utama** | PostgreSQL | Digunakan untuk RBAC dan data relasional. |
| **Database Dinamis** | MongoDB | Digunakan untuk Data Prestasi Dinamis. |
| **Keamanan** | JWT | Digunakan untuk *token-based authentication*. |
| **Dokumentasi** | Swagger | Direncanakan untuk dokumentasi API. |

---

### API Endpoints Kunci

| Endpoint | Metode | Deskripsi | Aktor |
| :--- | :--- | :--- | :--- |
| `/api/v1/auth/login` | `POST` | Autentikasi pengguna. | Semua Role |
| `/api/v1/users` | `GET/POST` | Manajemen pengguna (Admin). | Admin |
| `/api/v1/achievements` | `POST` | Create/Submit prestasi baru. | Mahasiswa |
| `/api/v1/achievements/:id/submit`| `POST` | Mengubah status prestasi menjadi `submitted`. | Mahasiswa |
| `/api/v1/achievements/:id/verify`| `POST` | Verifikasi prestasi (Status menjadi `verified`). | Dosen Wali |
| `/api/v1/achievements/:id/reject`| `POST` | Menolak prestasi (Status menjadi `rejected`). | Dosen Wali |
| `/api/v1/reports/statistics` | `GET` | Mendapatkan statistik prestasi. | Semua Role |

---

### Karakteristik Pengguna dan Hak Akses

| Role | Deskripsi | Hak Akses Utama |
| :--- | :--- | :--- |
| **Admin** | Pengelola sistem. | *Full access* ke semua fitur. |
| **Mahasiswa** | Pelapor prestasi. | *Create, read, update* prestasi sendiri. |
| **Dosen Wali** | Verifikator prestasi. | *Read, verify* prestasi mahasiswa bimbingannya. |

---

### Kode Kesalahan (Error Codes) API

| Code | Message | Description |
| :--- | :--- | :--- |
| **400** | Bad Request | Invalid input data. |
| **401** | Unauthorized | Missing or invalid token. |
| **403** | Forbidden | Insufficient permissions (RBAC check failed). |
| **404** | Not Found | Resource not found. |
| **409** | Conflict | Duplicate entry. |
| **422** | Unprocessable Entity | Validation error. |
| **500** | Internal Server Error | Server error. |

---

### Rencana Tambahan

* **Pengujian:** Menggunakan **Unit Testing** untuk menguji fungsi dan *method* individual.
* **Repositori:** Proyek akan menggunakan **Github Repository**.

---

**[Tambahkan instruksi *setup* lingkungan lokal di sini]**

---

Apakah draf *README* ini sudah sesuai dengan yang Anda inginkan?