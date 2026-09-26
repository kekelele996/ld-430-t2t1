# AssetHub 数字素材库与共享 API 服务

面向设计师和创意团队的数字素材（图片/矢量/字体/模板）上传、分类管理、搜索、标签与权限共享的纯后端 RESTful API 服务。

## Docker 快速启动（推荐）

```bash
cp .env.example .env
docker compose up -d --build
docker compose ps
```

启动后访问地址：

- 后端 API：http://localhost:19310
- 健康检查：http://localhost:19310/healthz
- 就绪检查：http://localhost:19310/readyz
- Swagger 文档：http://localhost:19310/api-docs
- MinIO Console：http://localhost:39011（账号使用 `.env` 中 `MINIO_ACCESS_KEY` / `MINIO_SECRET_KEY`）

种子管理员账号（启动时自动创建）：`admin@assethub.local` / `AssethubAdmin2026`（可用 `.env` 中 `ADMIN_EMAIL` / `ADMIN_PASSWORD` 覆盖，生产环境务必替换）。

## 本地开发（备选）

```bash
cd backend
go mod tidy
go run ./cmd/server
```

本地运行时需自行提供 MongoDB、Redis、MinIO，并通过环境变量注入连接信息（参考 `.env.example`）。

## 项目主要功能

- 素材上传与元数据管理：文件存入 MinIO，生成文件 URL 与缩略图 URL，支持 Draft → Published → Archived 状态流转
- 素材分类管理：多级父子分类，树形结构返回
- 素材集/收藏夹：创建收藏夹、添加/移除素材、设置公开、协作成员共同维护
- 素材搜索与筛选：标题/描述/标签/类型/许可/状态组合筛选，分页与下载量/浏览量排序
- 下载记录与统计：每次下载留痕，统计下载/浏览次数，热度排行
- 标签管理：分类标签、使用次数统计、标签云
- 审核流程：Moderator/Admin 审核，Approved / Rejected / NeedsRevision
- 权限共享与 RBAC：Admin / Moderator / Uploader / Viewer，Commercial 许可素材需额外权限
- 团队共享凭据：向外部服务开放素材能力而无需交付员工登录凭据；管理员按 `assets:read`（只读素材）/`collections:write`（维护收藏夹）/`downloads:write`（登记下载）分别授权并设置每日调用上限；明文密钥仅在创建时返回一次，列表只展示名称、范围、额度与末四位；外部请求经 `X-Api-Key` 调用 `/shared/*`，越权 403、过期/撤销 401、额度用完 429，每次调用记录凭据、路径、结果与时间，撤销即时生效
- 审计日志：素材上传、审核、下载、许可变更全程留痕
- 接口限流：下载每 IP 每分钟 10 次、上传每 IP 每小时 20 次（Redis 计数）

## 技术栈

| 层 | 技术 |
| --- | --- |
| 后端 | Go 1.22 + Gin + mongo-driver（repository 模式）+ MinIO + Redis |

其他依赖：golang-jwt/jwt/v5、go-playground/validator/v10、gin-contrib/cors、caarlos0/env/v11、log/slog、go-redis/v9、minio-go/v7。

## 项目目录结构

```
├── backend/
│   ├── cmd/server/main.go          # 装配依赖、启动服务
│   └── internal/
│       ├── config/                 # 环境变量配置（config/mongo/jwt/redis/minio）
│       ├── model/                  # MongoDB 文档模型
│       ├── repository/             # mongo-driver repository 模式
│       ├── service/                # 业务逻辑层（含 MinIO storage、搜索）
│       ├── handler/                # HTTP 处理器
│       ├── router/                 # Gin 路由注册（/api/v1）
│       ├── middleware/             # auth/rbac/audit/error/rate_limiter/request_logger/validation
│       ├── dto/                    # 请求/响应结构体与校验
│       ├── constants/              # 枚举、错误码、消息、日志模板
│       ├── errors/                 # 统一错误类型
│       ├── util/                   # jwt/logger/response/file_validator/thumbnail/password
│       └── client/                 # mongo/redis/minio 客户端
│   └── database/seeds/             # 种子数据（角色、分类、标签、管理员）
├── api/openapi.json                # OpenAPI 文档（/swagger/doc.json）
├── docker-compose.yml
└── .env / .env.example
```

## 主要 API 列表（前缀 `/api/v1`）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| POST | /auth/register | 注册（默认 Uploader） |
| POST | /auth/login | 登录，返回 JWT |
| GET | /auth/me | 当前用户信息 |
| POST | /assets/upload | 上传素材（限流 20 次/小时/IP） |
| GET | /assets | 素材搜索/筛选/分页 |
| GET | /assets/hot | 热度排行 |
| GET/PUT | /assets/:id | 详情 / 编辑 |
| POST | /assets/:id/publish | 发布 |
| POST | /assets/:id/archive | 下架 |
| POST | /assets/:id/download | 下载（限流 10 次/分钟/IP） |
| POST/GET | /assets/:id/reviews | 提交/查看审核 |
| GET/POST | /categories | 分类树 / 创建分类 |
| GET/POST | /tags | 标签云 / 创建标签 |
| GET/POST | /collections | 收藏夹列表 / 创建 |
| POST | /collections/:id/assets | 收藏夹添加素材 |
| DELETE | /collections/:id/assets/:assetId | 收藏夹移除素材 |
| POST | /collections/:id/members | 添加协作成员 |
| GET | /downloads | 下载记录 |
| GET/POST | /shared-credentials | 共享凭据列表 / 签发（Admin） |
| DELETE | /shared-credentials/:id | 撤销共享凭据（Admin，立即生效） |
| GET | /shared-credentials/:id/call-logs | 凭据调用记录（Admin） |
| GET | /shared/assets、/shared/assets/hot、/shared/assets/:id | 外部共享：只读已发布素材（需 assets:read） |
| POST | /shared/assets/:id/download | 外部共享：登记下载（需 downloads:write） |
| GET/POST/PUT | /shared/collections 及 /assets 子路由 | 外部共享：维护团队收藏夹（需 collections:write） |
| GET | /audit-logs | 审计日志（Admin） |

统一响应格式：`{ "code": 0, "message": "ok", "data": ... }`。

### 团队共享凭据用法

管理员通过 JWT 登录后签发凭据（`scopes` 可多选，`daily_limit` 为每日调用上限，`expires_at` 可选）：

```bash
curl -X POST http://localhost:19310/api/v1/shared-credentials \
  -H "Authorization: Bearer <管理员 JWT>" -H "Content-Type: application/json" \
  -d '{"name":"外部设计平台","scopes":["assets:read","downloads:write"],"daily_limit":1000,"expires_at":"2027-01-01T00:00:00Z"}'
```

响应中的 `key`（形如 `ahk_...`）**仅展示这一次**，服务端只存 SHA-256 哈希与末四位。外部服务请求时在 `X-Api-Key` 头携带：

```bash
curl http://localhost:19310/api/v1/shared/assets -H "X-Api-Key: ahk_xxxx"
```

返回码约定：凭据不存在/过期/已撤销 → `401`；凭据有效但缺少所需范围 → `403`（越权不消耗当日额度）；当日调用超过 `daily_limit` → `429`。撤销（`DELETE /shared-credentials/:id`）后新请求立即被拒绝。每次调用（含拒绝）都会按凭据记录方法、路径、状态码、结果与时间，可在 `GET /shared-credentials/:id/call-logs` 查看。

枚举定义位置：`backend/internal/constants/enums.go`（AssetType / LicenseType / AssetStatus / ReviewResult）。

## 环境变量说明

| 变量 | 说明 | 默认值 |
| --- | --- | --- |
| COMPOSE_PROJECT_NAME | Compose 项目名 | assethub |
| BACKEND_PORT | 后端宿主端口 | 19310 |
| MONGO_PORT | MongoDB 宿主端口 | 33330 |
| REDIS_PORT | Redis 宿主端口 | 36310 |
| MINIO_API_PORT | MinIO API 宿主端口 | 39010 |
| MINIO_CONSOLE_PORT | MinIO Console 宿主端口 | 39011 |
| MONGO_URI / MONGO_DATABASE | MongoDB 连接 | mongodb://mongodb:27017/asset_library |
| REDIS_URL | Redis 连接 | redis://redis:6379/0 |
| JWT_SECRET | JWT 签名密钥（生产至少 32 字符） | assethub-dev-secret-please-change-7f3a9c1e |
| CORS_ALLOWED_ORIGINS | 跨域来源（生产禁止 *） | http://localhost:19310 |
| MINIO_ACCESS_KEY / MINIO_SECRET_KEY / MINIO_BUCKET | MinIO 凭证与桶 | assethub-access / assethub-secret-change-me-8d2e / assethub |
| ADMIN_EMAIL / ADMIN_PASSWORD | 种子管理员账号 | admin@assethub.local / AssethubAdmin2026 |

## Docker 部署说明

- 服务编排：MongoDB 7 + Redis 7 + MinIO + 后端，全部使用命名卷持久化
- 后端内部端口固定 8080，映射到宿主 `${BACKEND_PORT}`
- MongoDB / MinIO 配置 healthcheck，后端 `depends_on` 等待数据库就绪
- 数据卷：`mongo_data`、`redis_data`、`minio_data`
- 停止与清理：`docker compose down -v --remove-orphans`

## License

MIT
