#!/bin/bash
# Конфигурация
APP_DIR="${GITHUB_WORKSPACE}/cmd/api"
OUTPUT_BINARY="app"
LOG_DIR="${GITHUB_WORKSPACE}/logs"
LOG_FILE="${LOG_DIR}/app.log"

echo "Initializing build process..."
echo "Current directory: $(pwd)"
ls -la

echo "Building application..."
cd "${APP_DIR}"
go mod tidy
go build -v -o "${OUTPUT_BINARY}" main.go
chmod +x "${OUTPUT_BINARY}"

echo "Build artifacts:"
ls -la "${OUTPUT_BINARY}"

echo "Preparing log directory..."
mkdir -p "${LOG_DIR}"
touch "${LOG_FILE}"

echo "Starting application service..."
nohup ./"${OUTPUT_BINARY}" > "${LOG_FILE}" 2>&1 &
APP_PID=$!

echo "Application successfully started with PID: ${APP_PID}"
echo "Logs are being written to: ${LOG_FILE}"
echo "Tail logs command: tail -f ${LOG_FILE}"

# Небольшая задержка для проверки старта
sleep 2
echo "Checking application status..."
if ps -p ${APP_PID} > /dev/null; then
    echo "Application process is running"
else
    echo "Application failed to start!"
    echo "Last log entries:"
    tail -n 20 "${LOG_FILE}"
    exit 1
fi

echo "Server info:"
echo "Go version: $(go version)"
echo "Go environment:"
go env