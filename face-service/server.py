"""
Face matching gRPC microservice for SkinTrader KYC verification.
Uses face_recognition library (dlib-based) for face detection and comparison.
"""

import os
import io
import logging
from concurrent import futures

import grpc
import numpy as np
from PIL import Image
import face_recognition

import face_match_pb2
import face_match_pb2_grpc

logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(message)s")
logger = logging.getLogger(__name__)

MATCH_THRESHOLD = float(os.getenv("FACE_MATCH_THRESHOLD", "0.6"))
MAX_IMAGE_SIZE = int(os.getenv("MAX_IMAGE_SIZE_MB", "10")) * 1024 * 1024


def load_image(image_bytes: bytes) -> np.ndarray:
    """Load image bytes into a numpy array for face_recognition."""
    if len(image_bytes) > MAX_IMAGE_SIZE:
        raise ValueError(f"Image too large: {len(image_bytes)} bytes")
    img = Image.open(io.BytesIO(image_bytes))
    if img.mode != "RGB":
        img = img.convert("RGB")
    return np.array(img)


class FaceMatchServicer(face_match_pb2_grpc.FaceMatchServiceServicer):
    """Implements the FaceMatchService gRPC service."""

    def CompareFaces(self, request, context):
        """Compare faces in two images and return match confidence."""
        logger.info("CompareFaces request received")

        try:
            id_image = load_image(request.id_image)
            selfie_image = load_image(request.selfie_image)
        except Exception as e:
            logger.error(f"Failed to load images: {e}")
            return face_match_pb2.CompareFacesResponse(
                match=False, confidence=0.0, error=f"Failed to load images: {e}"
            )

        # Detect faces in ID document
        id_encodings = face_recognition.face_encodings(id_image)
        if len(id_encodings) == 0:
            logger.warning("No face detected in ID document")
            return face_match_pb2.CompareFacesResponse(
                match=False, confidence=0.0, error="No face detected in ID document"
            )
        if len(id_encodings) > 1:
            logger.warning(f"Multiple faces ({len(id_encodings)}) in ID document, using first")

        # Detect faces in selfie
        selfie_encodings = face_recognition.face_encodings(selfie_image)
        if len(selfie_encodings) == 0:
            logger.warning("No face detected in selfie")
            return face_match_pb2.CompareFacesResponse(
                match=False, confidence=0.0, error="No face detected in selfie"
            )
        if len(selfie_encodings) > 1:
            logger.warning(f"Multiple faces ({len(selfie_encodings)}) in selfie, using first")

        # Compare faces
        distance = face_recognition.face_distance([id_encodings[0]], selfie_encodings[0])[0]
        confidence = max(0.0, 1.0 - distance)
        is_match = confidence >= MATCH_THRESHOLD

        logger.info(f"Face comparison: distance={distance:.4f}, confidence={confidence:.4f}, match={is_match}")

        return face_match_pb2.CompareFacesResponse(
            match=is_match, confidence=confidence, error=""
        )

    def DetectFace(self, request, context):
        """Detect faces in an image."""
        logger.info("DetectFace request received")

        try:
            image = load_image(request.image)
        except Exception as e:
            logger.error(f"Failed to load image: {e}")
            return face_match_pb2.DetectFaceResponse(
                face_detected=False, face_count=0, error=f"Failed to load image: {e}"
            )

        face_locations = face_recognition.face_locations(image)
        face_count = len(face_locations)

        logger.info(f"Face detection: found {face_count} face(s)")

        return face_match_pb2.DetectFaceResponse(
            face_detected=face_count > 0, face_count=face_count, error=""
        )


def serve():
    port = os.getenv("FACE_SERVICE_PORT", "50051")
    max_workers = int(os.getenv("FACE_SERVICE_WORKERS", "4"))

    server = grpc.server(
        futures.ThreadPoolExecutor(max_workers=max_workers),
        options=[
            ("grpc.max_receive_message_length", MAX_IMAGE_SIZE * 2),
            ("grpc.max_send_message_length", MAX_IMAGE_SIZE * 2),
        ],
    )
    face_match_pb2_grpc.add_FaceMatchServiceServicer_to_server(
        FaceMatchServicer(), server
    )
    server.add_insecure_port(f"[::]:{port}")
    server.start()
    logger.info(f"Face matching service started on port {port} with {max_workers} workers")
    server.wait_for_termination()


if __name__ == "__main__":
    serve()
