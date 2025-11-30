

## README Proyek: Sistem Pelaporan Prestasi Mahasiswa

### Gambaran Umum Proyek

[cite_start]**Sistem Pelaporan Prestasi Mahasiswa** adalah aplikasi *back-end* berbasis **REST API** yang dirancang untuk mengelola proses pelaporan, verifikasi, dan pengelolaan data prestasi akademik dan non-akademik mahasiswa[cite: 22]. [cite_start]Sistem ini mendukung arsitektur *multi-role* dan menyediakan fleksibilitas data dengan dukungan *field* prestasi yang dinamis[cite: 22].

---

### Fitur Utama (Functionalitas)

* [cite_start]**Autentikasi & Otorisasi:** Pengguna dapat *login* menggunakan *username/email* dan *password*[cite: 161, 162]. [cite_start]Setiap *endpoint* dilindungi dengan mekanisme **Role-Based Access Control (RBAC)** menggunakan **JSON Web Token (JWT)**[cite: 167, 169, 170].
* [cite_start]**Manajemen Prestasi Mahasiswa:** Mahasiswa dapat menambahkan, *read*, dan *update* prestasi mereka sendiri[cite: 30].
* [cite_start]**Workflow Prestasi:** Mahasiswa dapat mengubah status prestasi dari **'draft'** menjadi **'submitted'** untuk diverifikasi[cite: 188, 195].
* [cite_start]**Verifikasi Prestasi (Dosen Wali):** Dosen Wali dapat melihat daftar prestasi mahasiswa bimbingannya[cite: 207]. [cite_start]Mereka dapat memverifikasi (**'verified'**) [cite: 219] [cite_start]atau menolak (**'rejected'**) [cite: 230] [cite_start]prestasi yang berstatus **'submitted'**[cite: 215, 225].
* [cite_start]**Manajemen Pengguna (Admin):** Admin memiliki *full access* untuk *CRUD users* dan menetapkan *roles*[cite: 30, 235, 239].
* [cite_start]**Pelaporan dan Analitik:** Mampu menghasilkan statistik prestasi, seperti total per tipe, per periode, top mahasiswa berprestasi, dan distribusi tingkat kompetisi[cite: 251, 254].

---

### Teknologi yang Digunakan

| Kategori | Teknologi | Deskripsi |
| :--- | :--- | :--- |
| **Arsitektur** | REST API | [cite_start]Aplikasi *back-end* berbasis Representational State Transfer API[cite: 22]. |
| **Database Utama** | PostgreSQL | [cite_start]Digunakan untuk RBAC dan data relasional[cite: 34]. |
| **Database Dinamis** | MongoDB | [cite_start]Digunakan untuk Data Prestasi Dinamis[cite: 106]. |
| **Keamanan** | JWT | [cite_start]Digunakan untuk *token-based authentication*[cite: 16]. |
| **Dokumentasi** | Swagger | [cite_start]Direncanakan untuk dokumentasi API[cite: 299]. |

---

### API Endpoints Kunci

| Endpoint | Metode | Deskripsi | Aktor |
| :--- | :--- | :--- | :--- |
| `/api/v1/auth/login` | `POST` | Autentikasi pengguna. | [cite_start]Semua Role [cite: 162] |
| `/api/v1/users` | `GET/POST` | [cite_start]Manajemen pengguna (Admin)[cite: 264, 266]. | [cite_start]Admin [cite: 236] |
| `/api/v1/achievements` | `POST` | [cite_start]Create/Submit prestasi baru[cite: 276]. | [cite_start]Mahasiswa [cite: 179] |
| `/api/v1/achievements/:id/submit`| `POST` | [cite_start]Mengubah status prestasi menjadi `submitted`[cite: 277, 195]. | [cite_start]Mahasiswa [cite: 189] |
| `/api/v1/achievements/:id/verify`| `POST` | [cite_start]Verifikasi prestasi (Status menjadi `verified`)[cite: 278, 219]. | [cite_start]Dosen Wali [cite: 214] |
| `/api/v1/achievements/:id/reject`| `POST` | [cite_start]Menolak prestasi (Status menjadi `rejected`)[cite: 279, 230]. | [cite_start]Dosen Wali [cite: 224] |
| `/api/v1/reports/statistics` | `GET` | [cite_start]Mendapatkan statistik prestasi[cite: 290]. | [cite_start]Semua Role [cite: 253] |

---

### Karakteristik Pengguna dan Hak Akses

| Role | Deskripsi | Hak Akses Utama |
| :--- | :--- | :--- |
| **Admin** | Pengelola sistem. | [cite_start]*Full access* ke semua fitur[cite: 30]. |
| **Mahasiswa** | Pelapor prestasi. | [cite_start]*Create, read, update* prestasi sendiri[cite: 30]. |
| **Dosen Wali** | Verifikator prestasi. | [cite_start]*Read, verify* prestasi mahasiswa bimbingannya[cite: 30]. |

---

### Kode Kesalahan (Error Codes) API

| Code | Message | Description |
| :--- | :--- | :--- |
| **400** | Bad Request | [cite_start]Invalid input data[cite: 346]. |
| **401** | Unauthorized | [cite_start]Missing or invalid token[cite: 346]. |
| **403** | Forbidden | [cite_start]Insufficient permissions (RBAC check failed)[cite: 346]. |
| **404** | Not Found | [cite_start]Resource not found[cite: 346]. |
| **409** | Conflict | [cite_start]Duplicate entry (Contoh: *username* atau *email* sudah terdaftar)[cite: 346]. |
| **422** | Unprocessable Entity | [cite_start]Validation error (Data tidak valid)[cite: 346]. |
| **500** | Internal Server Error | [cite_start]Server error[cite: 346]. |

---

### Rencana Tambahan

* [cite_start]**Pengujian:** Menggunakan **Unit Testing** untuk menguji fungsi dan *method* individual[cite: 295, 296].
* [cite_start]**Repositori:** Proyek akan menggunakan **Github Repository**[cite: 300].

