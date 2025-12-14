-- Aktifkan ekstensi UUID (jika belum aktif)
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 1. Tabel Roles
CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(50) UNIQUE NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 2. Tabel Users
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(100) NOT NULL,
    role_id UUID REFERENCES roles(id),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 3. Tabel Permissions & Role Permissions
CREATE TABLE permissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) UNIQUE NOT NULL,
    resource VARCHAR(50) NOT NULL,
    action VARCHAR(50) NOT NULL,
    description TEXT
);

CREATE TABLE role_permissions (
    role_id UUID REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

-- 4. Tabel Lecturers (Dosen)
CREATE TABLE lecturers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    lecturer_id VARCHAR(20) UNIQUE NOT NULL,
    department VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 5. Tabel Students (Mahasiswa)
CREATE TABLE students (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    student_id VARCHAR(20) UNIQUE NOT NULL,
    program_study VARCHAR(100),
    academic_year VARCHAR(10),
    advisor_id UUID REFERENCES lecturers(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 6. Tabel Achievement References
CREATE TABLE achievement_references (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    mongo_achievement_id VARCHAR(24) NOT NULL,
    status VARCHAR(20) DEFAULT 'draft',
    submitted_at TIMESTAMP,
    verified_at TIMESTAMP,
    verified_by UUID REFERENCES users(id),
    rejection_note TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO roles (id, name, description, permission, created_at) VALUES
    (gen_random_uuid(), 'Admin', 'Pengelola sistem', 'Full access ke semua fitur', NOW()),
    (gen_random_uuid(), 'Mahasiswa', 'Pelapor prestasi', 'Create, read, update prestasi sendiri', NOW()),
    (gen_random_uuid(), 'Dosen Wali', 'Verifikator prestasi', 'Read, verify prestasi mahasiswa bimbingannya', NOW());

INSERT INTO permissions (id, name, resource, action, description) VALUES
    (gen_random_uuid(), 'achievement:create', 'achievement', 'create', 'Izin untuk membuat data prestasi'),
    (gen_random_uuid(), 'achievement:read', 'achievement', 'read', 'Izin untuk membaca data prestasi'),
    (gen_random_uuid(), 'achievement:update', 'achievement', 'update', 'Izin untuk mengubah data prestasi'),
    (gen_random_uuid(), 'achievement:delete', 'achievement', 'delete', 'Izin untuk menghapus data prestasi'),
    (gen_random_uuid(), 'achievement:verify', 'achievement', 'verify', 'Izin untuk memverifikasi prestasi mahasiswa'),
    (gen_random_uuid(), 'user:manage', 'user', 'manage', 'Izin untuk mengelola data pengguna');

INSERT INTO users (id, username, email, password_hash, full_name, role_id, is_active, created_at, updated_at) VALUES
(gen_random_uuid(), 'admin', 'admin@example.com', '$2a$10$Zmd9BmoZKt/aHkSFOtSd2e3qdQYZDzUre1pginyKmIoHfuJFOSsye', 'Administrator Sistem', '22a47ebc-192e-451e-90ab-0dbf6f4191cc', TRUE, NOW(), NOW()),
(gen_random_uuid(), 'mahasiswa1', 'mahasiswa1@example.com', '$2a$10$Zmd9BmoZKt/aHkSFOtSd2e3qdQYZDzUre1pginyKmIoHfuJFOSsye', 'Mahasiswa Satu', '1cf7200e-23e9-4782-af90-bff4404ab976', TRUE, NOW(), NOW()),
(gen_random_uuid(), 'dosenwali1', 'dosenwali1@example.com', '$2a$10$Zmd9BmoZKt/aHkSFOtSd2e3qdQYZDzUre1pginyKmIoHfuJFOSsye', 'Dosen Wali Akademik', '10018f1c-d1be-4a84-9003-bd988b0f56f3', TRUE, NOW(), NOW());

-- ============================================
-- ADMIN - Full Access
-- ============================================
INSERT INTO role_permissions (role_id, permission_id) VALUES
('22a47ebc-192e-451e-90ab-0dbf6f4191cc', 'e7222273-6262-462c-9e7a-e52f81ab0364'), -- create
('22a47ebc-192e-451e-90ab-0dbf6f4191cc', '960319f8-2a7f-4d8f-8a26-b8ee40ed17a9'), -- read
('22a47ebc-192e-451e-90ab-0dbf6f4191cc', '69b7f93b-6f9b-49a5-8ce0-6f6a35495ebd'), -- update
('22a47ebc-192e-451e-90ab-0dbf6f4191cc', '19606c2f-af3a-4bf6-9523-d774a5dc3e1f'), -- delete
('22a47ebc-192e-451e-90ab-0dbf6f4191cc', '10c037f5-e6fd-4387-b929-6771f884d452'), -- verify
('22a47ebc-192e-451e-90ab-0dbf6f4191cc', '834e9ef8-e2ba-4f22-8657-0ae4d15b832e'); -- user_manage


-- ============================================
-- MAHASISWA - Create, Read, Update
-- ============================================
INSERT INTO role_permissions (role_id, permission_id) VALUES
('1cf7200e-23e9-4782-af90-bff4404ab976', 'e7222273-6262-462c-9e7a-e52f81ab0364'), -- create
('1cf7200e-23e9-4782-af90-bff4404ab976', '960319f8-2a7f-4d8f-8a26-b8ee40ed17a9'), -- read
('1cf7200e-23e9-4782-af90-bff4404ab976', '69b7f93b-6f9b-49a5-8ce0-6f6a35495ebd'); -- update



-- ============================================
-- DOSEN WALI - Read, Verify
-- ============================================
INSERT INTO role_permissions (role_id, permission_id) VALUES
('10018f1c-d1be-4a84-9003-bd988b0f56f3', '960319f8-2a7f-4d8f-8a26-b8ee40ed17a9'), -- read
('10018f1c-d1be-4a84-9003-bd988b0f56f3', '10c037f5-e6fd-4387-b929-6771f884d452'); -- verify

INSERT INTO lecturers (id, user_id, lecturer_id, department, created_at) 
VALUES (gen_random_uuid(), '64d1bd23-3c0f-483a-9c54-a44d8c7a64a7', 'LEC1', 'Computer Science Department', NOW());

INSERT INTO students (id, user_id, student_id, program_study, academic_year, advisor_id, created_at) 
VALUES (gen_random_uuid(), 'cecf76ed-e30a-40ca-b602-a6a3d0f3f93f', 'NIM0012025', 'Information Systems', '2025/2026', '23506891-d6d1-4614-87d1-acbf62d0eada', NOW());

