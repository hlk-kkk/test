"""Health check endpoints."""

from fastapi import APIRouter

router = APIRouter()


@router.get("/health")
async def health_check():
    """Health check endpoint."""
    return {"status": "healthy", "service": "pictactic"}


@router.get("/")
async def root():
    """Root endpoint."""
    return {
        "name": "PicTacticAgent API",
        "version": "0.1.0",
        "status": "running",
    }
