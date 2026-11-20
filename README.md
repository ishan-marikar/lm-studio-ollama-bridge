# lm-studio-ollama-bridge

> I'm refactoring the code to make it more modular and easier to maintain. The current codebase is a bit messy and needs some cleanup. I'll be updating the README with the new instructions and features once the refactoring is complete. You can follow the progress in the [next branch](https://github.com/ishan-marikar/lm-studio-ollama-bridge/tree/next). Stay tuned!

`lm-studio-ollama-bridge` is a standalone Go-based utility that synchronizes your local Ollama models with external applications such as LM Studio. Inspired by the original [matts-shell-scripts/syncmodels](https://github.com/technovangelist/matts-shell-scripts/blob/main/syncmodels) project , `lm-studio-ollama-bridge` extends its functionality by making it more performant and portable (not that it needs to be but why not?).

This [video demonstration](https://www.youtube.com/watch?v=UfhXbwA5thQ) by [Matt Williams](https://www.youtube.com/@technovangelist) (a founding maintainer of Ollama) walks through the synchronization workflow and highlights how [matts-shell-scripts/syncmodels](https://github.com/technovangelist/matts-shell-scripts/blob/main/syncmodels) integrates Ollama models with LM Studio and other tools.

---

## Overview

`lm-studio-ollama-bridge` automates the process of exposing your locally managed Ollama models in a format that external applications can readily consume. It performs the following key tasks:

- **Manifest Scanning:** Recursively searches for model manifest JSON files in your local Ollama installation.
- **JSON Parsing:** Unmarshals both the manifest and associated model configuration JSON (from the blobs directory) to extract critical metadata such as model type, file type, and format.
- **Symlink Creation:** Generates descriptive symbolic links that point from a public directory (used by LM Studio) to the actual model blob files.

This streamlined workflow ensures that your models remain up to date and accessible by any downstream tools.

---

## Features

- **Independent Go Implementation:** A compiled binary for improved performance and portability.
- **Automatic Directory Management:** Creates required directories and removes stale symbolic links automatically.
- **Configurable Naming Conventions:** Constructs symlink filenames based on model name, type, file type, and format.
- **Digest Consistency:** Normalizes digest strings to ensure reliable file referencing.

---

## Prerequisites

Before using `lm-studio-ollama-bridge`, make sure you have:

- **Go:** Version 1.16 or later installed.
- **Git:** For cloning the repository.
- **Local Ollama Installation:** Models should (and will usually) reside under your home directory at `~/.ollama/models` with manifests in `manifests/registry.ollama.ai` and blobs in `blobs`.
- **Proper Permissions:** Ability to create directories and symbolic links (the default destination is `~/.cache/lm-studio/models/ollama` but this can be configured).

---

## Installation

1. **Clone the Repository:**

   ```bash
   git clone https://github.com/ishan-marikar/lm-studio-ollama-bridge  .git
   cd lm-studio-ollama-bridge
   ```

2. **Build the Binary:**

   Ensure your Go environment is set up, then run:

   ```bash
   go mod tidy
   go build -o lm-studio-ollama-bridge
   ```

This compiles the project into an executable binary named `lm-studio-ollama-bridge`.

---

## Configuration

By default, `lm-studio-ollama-bridge` uses the following directory structure:

- **Manifest Directory:** `~/.ollama/models/manifests/registry.ollama.ai`
- **Blob Directory:** `~/.ollama/models/blobs`
- **Destination Directory (LM Studio):** `~/.cache/lm-studio/models/ollama`

If your environment differs from these defaults, you can modify the corresponding paths in the source code (inside the `main` function) to suit your setup.

---

## Usage

Once built, simply run the executable:

```bash
./lm-studio-ollama-bridge
```

The tool will:

1. **Scan:** Locate all manifest files within your local Ollama manifests directory.
2. **Process:** For each manifest, extract model details and identify the associated model blob.
3. **Link:** Create a symlink in the LM Studio directory using the naming convention:

   ```
   <model-name>-<model-type>-<file-type>.<model-format>
   ```

4. **Prune:** Remove links it made earlier that no current manifest accounts for (symlinks into the
   Ollama blob directory, e.g. for a deleted tag whose blob another tag still uses). Links to other
   places and regular files are never touched, and nothing is pruned if any manifest failed to process.
5. **Log:** Output detailed logs about the processing of each manifest, including any errors.

After running, your LM Studio models directory will be updated with the latest symbolic links, ensuring that external applications have immediate access to your models.

---

## Contributing

Contributions are highly welcome! If you have suggestions, bug reports, or feature requests:

- **Open an Issue:** Use the GitHub issue tracker.
- **Submit a Pull Request:** Please follow the repository’s contribution guidelines, ensuring that your changes are well documented and tested.

---

## License

`lm-studio-ollama-bridge` is released under the MIT License. See the [LICENSE](./LICENSE) file for more details.

---

## Acknowledgements

- **Inspiration:** This project draws significant inspiration from [matts-shell-scripts/syncmodels](https://github.com/technovangelist/matts-shell-scripts/blob/main/syncmodels) .
- **Video Demonstration:** The [Sync Ollama Models with Other Tools](https://www.youtube.com/watch?v=UfhXbwA5thQ) video provided valuable insights into the workflow.
- **Ollama Community:** Thanks to the community for pioneering local model management and sharing ideas that helped shape this project.
