package h

type TokenReviewInterface interface {
	TokenReviewExpansion
}

type TokenReviewExpansion interface {
	Create() (err error)
}

