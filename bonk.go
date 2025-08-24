// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package bonk

// Generate proto code
//go:generate go tool buf generate

// Docs
//go:generate ./scripts/gomarkdoc.sh ./api/go
//go:generate go run -tags docs ./cmd/bonk docs
