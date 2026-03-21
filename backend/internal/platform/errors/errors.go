package errors

const (
	CodeBadRequest         = "BAD_REQUEST"
	CodeUnauthorized       = "AUTH_UNAUTHORIZED"
	CodeForbidden          = "AUTH_FORBIDDEN"
	CodeTripNotFound       = "TRIP_NOT_FOUND"
	CodeVersionConflict    = "TRIP_VERSION_CONFLICT"
	CodeTimeConflict       = "ITINERARY_TIME_CONFLICT"
	CodeInvalidDateRange   = "ITINERARY_INVALID_DATE_RANGE"
	CodeBudgetCurrency     = "BUDGET_INVALID_CURRENCY"
	CodeMapProviderTimeout = "MAP_PROVIDER_TIMEOUT"
	CodeAIProviderTimeout  = "AI_PROVIDER_TIMEOUT"
	CodeRateLimitExceeded  = "RATE_LIMIT_EXCEEDED"
	CodeInternalError      = "INTERNAL_ERROR"
	CodeNotImplemented     = "NOT_IMPLEMENTED"
)
