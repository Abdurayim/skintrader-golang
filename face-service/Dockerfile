FROM python:3.11-slim

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    cmake build-essential libopenblas-dev liblapack-dev libx11-dev && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

# Generate gRPC stubs from proto
COPY ../proto/face_match/face_match.proto /tmp/face_match.proto
RUN python -m grpc_tools.protoc \
    -I/tmp \
    --python_out=. \
    --grpc_python_out=. \
    /tmp/face_match.proto

ENV FACE_SERVICE_PORT=50051
ENV FACE_MATCH_THRESHOLD=0.6
ENV FACE_SERVICE_WORKERS=4

EXPOSE 50051

CMD ["python", "server.py"]
