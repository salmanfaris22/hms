package repository

// Upload module currently has no persistence layer — signing is stateless.
// File kept for architectural symmetry.

type Repository struct{}

func New() *Repository { return &Repository{} }
