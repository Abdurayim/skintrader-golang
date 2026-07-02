#!/bin/bash
# Generate Python gRPC stubs from proto definition
# Run from the face-service directory

PROTO_DIR="../proto/face_match"
OUT_DIR="."

python -m grpc_tools.protoc \
    -I"$PROTO_DIR" \
    --python_out="$OUT_DIR" \
    --grpc_python_out="$OUT_DIR" \
    "$PROTO_DIR/face_match.proto"

echo "Generated face_match_pb2.py and face_match_pb2_grpc.py"
