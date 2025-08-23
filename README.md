# BONK
*Your favorite build system's favorite build system.*

---

Bonk is a build system _framework_, designed to be extensible and unopinionated.
A complete bonk system consits of a plugin-provided front-end, the bonk task
scheduler, and a series of plugin-provided backends. Conceptually, bonk could
replace anything and everything from GNU make to package managers.

### How it works

Executing a bonk build takes the following steps:

1. bonk reads your project file (bonk.cue, bonk.yaml, bonk.json, etc.).
1. bonk spawns tasks to initialize any required plugins.
1. bonk sends your project configuration to configured front-end,
which returns a series of tasks.
1. bonk schedules the tasks according to dependencies, and skips any tasks that are up to date.
1. bonk sends the tasks to the registered plugin-provided backends.

### How to use bonk

You can't yet, I'm still working on it.
