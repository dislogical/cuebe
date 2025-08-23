// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package bonk

// Generate proto code
//go:generate go tool buf generate

// Docs
//go:generate go tool gomarkdoc go.bonk.build/api/go
//go:generate go run -tags docs ./cmd/bonk docs
