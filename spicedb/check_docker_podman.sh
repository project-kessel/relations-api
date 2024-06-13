# Function to check if a command is available
command_exists() {
  command -v "$1" >/dev/null 2>&1
}

# Check if Docker is installed
if command_exists docker; then
  echo "Docker is installed. Using Docker."
  export DOCKER=docker
else
  echo "Docker not found. Checking for Podman."

  # Check if Podman is installed
  if command_exists podman; then
    echo "Podman is installed. Using Podman."
    export DOCKER=podman
  else
    echo "Podman not found. Please install either Docker or Podman to proceed."
    exit 1
  fi
fi

