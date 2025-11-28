# Image Processor

Service for asynchronous image processing with Kafka-based task queue. Supports resize, thumbnail generation, and watermark application.

## Features

- **Resize** - Scale images to 800x600 with aspect ratio preservation
- **Thumbnail** - Generate 200x150 thumbnails with aspect ratio preservation
- **Watermark** - Apply large red watermark text across images
- **Async Processing** - Kafka-based queue for background processing
- **REST API** - Upload, retrieve, and manage images
- **Web UI** - Simple interface for image upload and viewing

## Tech Stack

- **Go 1.24** - Backend services
- **PostgreSQL** - Metadata storage
- **Kafka** - Message queue
- **Docker** - Containerization
- **WBF Framework** - Configuration and utilities


Resize image                     | Thumbnail image                            | Watermark image
-----------------------------------|----------------------------------------|---------------------------------------
![srcImage](test_data/resize.jpg) | ![dstImage](test_data/thumbnail.jpg) | ![dstImage](test_data/watermark.jpg)

## Quick Start
```bash
# Clone repository
git clone https://github.com/yokitheyo/imageprocessor.git
cd wb_level_3_04

# Start services (dev-mode, default)
docker compose up -d

# Access web UI
open http://localhost:8080
```

## API Endpoints

- `POST /upload` - Upload image with processing type (resize/thumbnail/watermark)
- `GET /images` - List all images
- `GET /image/:id` - Get processed image
- `GET /image/:id/original` - Get original image
- `DELETE /image/:id` - Delete image

## Project Structure
```
.
├── cmd/
│   ├── api/          # API server entry point
│   └── worker/       # Worker service entry point
├── internal/
│   ├── config/       # Configuration management (wbf integration)
│   ├── domain/       # Business entities and interfaces
│   ├── dto/          # Data transfer objects
│   ├── handler/      # HTTP handlers and middleware
│   ├── infrastructure/
│   │   ├── database/ # PostgreSQL migrations
│   │   ├── kafka/    # Kafka producer/consumer
│   │   ├── processor/# Image processing logic
│   │   └── storage/  # File storage (local/S3)
│   ├── repository/   # Database repositories
│   ├── usecase/      # Business logic
│   └── worker/       # Task handlers
├── migrations/       # SQL migrations
├── static/           # Web UI assets
├── storage/          # Local file storage
│   ├── original/     # Original uploads
│   └── processed/    # Processed images
├── config.yaml       # Application configuration
└── docker-compose.yml
```

# View logs
docker logs -f imageprocessor_api
docker logs -f imageprocessor_worker

# Stop services
docker-compose down
```

## Architecture

┌─────────┐      ┌─────────┐      ┌──────────┐
│ Client  │─────▶│   API   │─────▶│ Postgres │
└─────────┘      └─────────┘      └──────────┘
                      │
                      ▼
                 ┌────────┐
                 │ Kafka  │
                 └────────┘
                      │
                      ▼
                 ┌────────┐      ┌─────────┐
                 │ Worker │─────▶│ Storage │
                 └────────┘      └─────────┘

