// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

@extern(embed)
package workflows

import (
	"cue.dev/x/githubactions"
)

github: {
	// Embed the contents of all YAML workflow files.
	workflows: _ @embed(glob=.github/workflows/*.yaml)

	// Validate the contents of each embedded file against the relevant schema.
	workflows: [_]: githubactions.#Workflow
}
