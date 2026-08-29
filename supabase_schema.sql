-- Schema table cho Supabase (PostgreSQL) - Sử dụng schema s3go

-- Thiết lập schema s3go làm mặc định cho các câu lệnh bên dưới
CREATE SCHEMA IF NOT EXISTS s3go;
SET search_path TO s3go;

-- 1. Tạo bảng connections trong schema s3go
CREATE TABLE IF NOT EXISTS connections (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    access_key VARCHAR(255) NOT NULL,
    secret_key_encrypted TEXT NOT NULL,
    region VARCHAR(255) NOT NULL,
    bucket VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL
);

-- 2. Khôi phục dữ liệu từ connections.json cục bộ vào schema s3go (Seeding Data)
INSERT INTO connections (id, name, access_key, secret_key_encrypted, region, bucket, created_at)
VALUES 
(
    '94ba331d-f7d1-4780-b75c-797e26432591', 
    'amaze-s3-uat', 
    'AKIAW3MZIVC7RUMTDWGF', 
    '6690b1d0505a530c921e84bde92b07ac979a4803bf528e9634125cf52ee62507c59ac96189e432bbe6ea28ea99ec1d19c5b466a44ea61c385a9320e938a6237abf534756', 
    'ap-southeast-1', 
    'amaze-uat-sftp', 
    '2026-06-02 17:26:36.331962+07:00'
),
(
    '2654a58d-4266-45a6-9fd9-abeb16f15633', 
    'amaze-s3-dev', 
    'AKIA5WHZX6C25WT4PD2J', 
    'd4ac35cc29ce66830dec765815039280f108e5127b3bf49693afa68c7b0ca351de90137706da993f0bfc4a0129cf356e323f44acdf0c5621a3a79b3f1a3a7536f20477e8', 
    'ap-southeast-1', 
    '', 
    '2026-06-02 17:34:25.600682+07:00'
)
ON CONFLICT (id) DO NOTHING;
