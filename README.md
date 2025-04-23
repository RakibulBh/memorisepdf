# Unmemorizable

Unmemorizable is a web application that converts presentation slides into interactive quizzes and flashcards, helping users study more efficiently and effectively.

## Features

- **Instant Quiz Generation**: Upload presentations and convert them into quizzes in under a minute
- **Interactive Learning**: Dynamic quizzes and flashcards to enhance retention
- **Efficient Study Time**: Reduce study time by up to 98% with targeted learning
- **Higher Engagement**: 5x higher engagement compared to traditional study methods

## Technology Stack

- **Frontend**: Next.js, React, TailwindCSS, Framer Motion
- **Backend**: Go
- **Database**: TBD

## Project Structure

```
├── web/                    # Frontend application
│   ├── app/                # Next.js app directory
│   ├── components/         # React components
│   ├── public/             # Static assets
│   └── ...
├── internal/               # Go backend code
├── cmd/                    # Go entry points
└── ...
```

## Development Setup

### Prerequisites

- Node.js (v18+)
- Go (v1.21+)
- Docker and Docker Compose (optional)

### Frontend Setup

```bash
cd web
npm install
npm run dev
```

### Backend Setup

```bash
# Using Go directly
go run cmd/app/main.go

# Using Docker
docker-compose up
```

## API Documentation

TBD

## Deployment

TBD

## Adding New Features

When adding new features to the project:

1. Create a new branch from main
2. Implement your feature
3. Write appropriate tests
4. Submit a PR for review

## Performance Considerations

- The application uses optimized image formats and sizes
- Code splitting is implemented for faster loading
- Animation effects are optimized to prevent performance issues

## Troubleshooting

TBD

## License

This project is proprietary and confidential. Unauthorized copying of files, via any medium is strictly prohibited.
