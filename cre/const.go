package cre

const (
	DefaultMaxResponseSizeBytes = 5 * 1024 * 1024 // 5 MB
	ResponseBufferTooSmall      = "response buffer too small: the serialized capability response exceeds the maximum allowed size. Consider reducing the response payload size or increasing DefaultMaxResponseSizeBytes"
	// proto encoder outputs a map with these keys so that user payload can be easily extracted
	ReportMetadataHeaderLength = 109
)
