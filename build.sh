#!/bin/bash

# AI Cart WebSocket - Multi-platform Build Script

set -e

# Build configuration
APP_NAME="ai-cart-websocket"
VERSION=${VERSION:-"1.0.0"}
BUILD_DIR="dist"

# Target platforms
TARGETS=(
    "linux/amd64/"
    "windows/amd64/.exe"
    "darwin/amd64/"
    "darwin/arm64/"
)

echo "🚀 Building AI Cart WebSocket v${VERSION}"
echo "=============================================="

# Clean previous builds
if [ -d "$BUILD_DIR" ]; then
    echo "🧹 Cleaning previous builds..."
    rm -rf "$BUILD_DIR"
fi

# Create build directory
mkdir -p "$BUILD_DIR"

# Copy .env.example to build directory
echo "📄 Copying configuration files..."
if [ -f ".env.example" ]; then
    cp .env.example "$BUILD_DIR/.env.example"
else
    # Create rudimentary .env.example if missing
    echo "PORT=8080" > "$BUILD_DIR/.env.example"
    echo "API_SECRET=change_me" >> "$BUILD_DIR/.env.example"
fi

# Create README for distribution
cat > "$BUILD_DIR/README.txt" << 'EOF'
AI Cart WebSocket Service - Distribution Package
=============================================

Real-time WebSocket Push Service for AI Cart.

Quick Start:

1. Copy .env.example to .env and configure:
   - PORT=8080
   - RABBITMQ_URL (Optional)
   - API_SECRET (for HTTP API)

2. Run the program:
   - Linux/macOS: ./ai-cart-websocket-{platform}-{arch}
   - Windows: ai-cart-websocket-windows-amd64.exe

3. Service will verify public key on startup.

For detailed documentation, see:
https://github.com/rakibhoossain/ai-cart-websocket

EOF

# Build for each target platform
for target in "${TARGETS[@]}"; do
    # Parse target
    IFS='/' read -r os arch extension <<< "$target"
    
    # Set output filename
    output_name="${APP_NAME}-${os}-${arch}${extension}"
    output_path="${BUILD_DIR}/${output_name}"
    
    echo "🔨 Building for ${os}/${arch}..."
    
    # Set environment variables and build
    GOOS=$os GOARCH=$arch CGO_ENABLED=0 go build \
        -ldflags "-s -w -X main.Version=${VERSION}" \
        -o "$output_path" \
        .
    
    # Create platform-specific package
    platform_dir="${BUILD_DIR}/${APP_NAME}-${VERSION}-${os}-${arch}"
    mkdir -p "$platform_dir"
    
    # Copy binary
    cp "$output_path" "$platform_dir/"
    
    # Copy configuration files
    if [ -f ".env.example" ]; then
        cp .env.example "$platform_dir/"
    fi
    cp "$BUILD_DIR/README.txt" "$platform_dir/"
    
    # Create platform-specific start script
    if [ "$os" = "windows" ]; then
        cat > "$platform_dir/start.bat" << 'EOF'
@echo off
echo Starting AI Cart WebSocket Service...
echo Make sure you have configured .env file
echo Press Ctrl+C to stop the server
${output_name}
pause
EOF
    else
        cat > "$platform_dir/start.sh" << EOF
#!/bin/bash
echo "Starting AI Cart WebSocket Service..."
echo "Make sure you have configured .env file"
echo "Press Ctrl+C to stop the server"
./${output_name}
EOF
        chmod +x "$platform_dir/start.sh"
    fi
    
    # Create archive
    echo "📦 Creating archive for ${os}/${arch}..."
    cd "$BUILD_DIR"
    
    if [ "$os" = "windows" ]; then
        zip -r "${APP_NAME}-${VERSION}-${os}-${arch}.zip" "${APP_NAME}-${VERSION}-${os}-${arch}/" > /dev/null
    else
        tar -czf "${APP_NAME}-${VERSION}-${os}-${arch}.tar.gz" "${APP_NAME}-${VERSION}-${os}-${arch}/"
    fi
    
    cd ..
    
    # Clean up directory (keep archive)
    rm -rf "$platform_dir"
    
    echo "✅ Built: ${output_name}"
done

# Create checksums
echo "🔐 Generating checksums..."
cd "$BUILD_DIR"
if command -v sha256sum >/dev/null 2>&1; then
    sha256sum *.tar.gz *.zip 2>/dev/null > checksums.txt || true
elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 *.tar.gz *.zip 2>/dev/null > checksums.txt || true
fi
cd ..

echo ""
echo "✅ Build completed successfully!"
echo "📁 Build artifacts are in the '${BUILD_DIR}' directory:"
echo ""
ls -la "$BUILD_DIR"
echo ""
echo "🚀 Ready for distribution!"