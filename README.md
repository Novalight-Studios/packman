# packman

## 📜 Description

This CLI was created to manage modpacks for [Modrinth](https://modrinth.com/) and [CurseForge](https://www.curseforge.com/minecraft) at the same time. <br />
Mods, resource packs and shaders are looked up on both platforms at once and stored in a single `packman.json` file. <br />
Easily create modpacks by placing your config files into an `overrides` directory, they will be added to your modpack when exporting it. <br />
Made for creators who publish the same modpack on Modrinth and CurseForge, without maintaining it twice.

## ✨ Features

- Create a new modpack
- Add mods, resource packs and shaders by slug (the type is detected automatically)
- Required dependencies are added along with the content that needs them
- Find the same project on both platforms, even when its slug is different (matched through the file itself, or with `--modrinth-slug` / `--curseforge-slug`)
- Remove mods, resource packs and shaders from the modpack
- Update everything to its latest version (dry run supported)
- Migrate the modpack to another Minecraft version or loader
- Export the modpack to a MRPACK file (Modrinth) and a ZIP file (CurseForge), (files are moved to: `dist`)
- Content available on one platform only is bundled as an override in the pack of the other one
- Every file of the `overrides` directory is added to both packs
- Generate a Markdown list of the modpack's content, with links to Modrinth or CurseForge
- Save your CurseForge API key once and for all

## 🔧 Commands

| Command                    | Description                                                                                                                 |
| -------------------------- | --------------------------------------------------------------------------------------------------------------------------- |
| `packman init`             | Create a new modpack in the current directory (`--name`, `--mc`, `--loader`, `--loader-version`, `--version`)               |
| `packman add <name>...`    | Add mods, resource packs or shaders (`--type`, `--source`, `--no-deps`, `--modrinth-slug`, `--curseforge-slug`)             |
| `packman remove <name>...` | Remove content from the modpack (alias: `rm`, `--type`)                                                                     |
| `packman update [name]...` | Update content to its latest version (`--type`, `--dry-run`, `--no-deps`)                                                   |
| `packman export`           | Export the modpack for Modrinth and CurseForge (`--format`, `--output`, `--skip-missing`)                                   |
| `packman list`             | List the content of the modpack as Markdown, with a link for each project (alias: `ls`, `--platform`, `--type`, `--output`) |
| `packman auth curseforge`  | Save your CurseForge API key (`--remove` to delete it)                                                                      |

Run `packman <command> --help` for the details of a command.

## 🌲 Modpack tree example

```
Modpack
┣ overrides
┃ ┣ config
┃ ┃ ┗ sodium-options.json
┃ ┣ icon.png
┃ ┗ options.txt
┣ dist
┃ ┣ my-modpack-1.0.0.mrpack
┃ ┗ my-modpack-1.0.0.zip
┗ packman.json
```

## 📄 packman.json

```json
{
  "name": "My Modpack",
  "version": "1.0.0",
  "mc_version": "1.21.1",
  "loader": "fabric",
  "loader_version": "0.20.1",
  "mods": [
    {
      "slug": "sodium",
      "name": "Sodium",
      "modrinth": {
        "projectID": "<project id>",
        "versionID": "<version id>",
        "path": "mods/sodium-fabric-<version>.jar",
        "hashes": {
          "sha1": "<sha1 hash>",
          "sha512": "<sha512 hash>"
        },
        "env": {
          "client": "required",
          "server": "unsupported"
        },
        "downloads": ["<url to download the mod>"],
        "fileSize": 123456
      },
      "curseforge": {
        "projectID": 394468,
        "fileID": 123456,
        "required": true,
        "isLocked": false
      }
    }
  ],
  "resources_packs": [],
  "shaders": []
}
```

Each entry has a `modrinth` and a `curseforge` block, so both export formats can be generated from the same file. A block is missing when the project doesn't exist on that platform.

## ⚙️ Requirements

- [Go 1.22 (or higher)](https://go.dev/)
- A [CurseForge API key](https://console.curseforge.com/) (optional, without it only Modrinth is used)

## 📦 Installation

### 👤 For users

```sh
go install github.com/Novalight-Studios/packman@latest
```

### 👨‍💻 For devs

```sh
git clone https://github.com/Novalight-Studios/packman.git
cd packman
```

Run the command without compiling:

```sh
go run . --help
```

To Compile:

```sh
make build
```

## 🚀 Usage

```sh
# Save your CurseForge API key (optional)
packman auth curseforge

# Create a modpack
packman init --name "My Modpack" --mc 1.21.1 --loader fabric

# Add content (mods, resource packs and shaders, with their required dependencies)
packman add sodium lithium iris complementary-reimagined

# Keep it up to date
packman update

# Export it to dist/
packman export

# List its content, with links to Modrinth
packman list -o PACK_CONTENT.md
```

### 🔑 CurseForge API key

`packman auth curseforge` asks for your key (hidden) and checks it before saving it.
It is stored in your user config directory, readable by you only, and **in plain text**:

- Windows: `%AppData%\packman\config.json`
- Linux: `~/.config/packman/config.json`
- macOS: `~/Library/Application Support/packman/config.json`

You can also use the `CURSEFORGE_API_KEY` environment variable, which takes priority over the saved key.
CurseForge keys contain `$` characters: put them between single quotes in your shell.

### ⚠️ Incompatible content

By default, `packman add` refuses content that has no version for your Minecraft version and loader. Use `--force` to add it anyway:

```sh
packman add some-mod --force
```

The newest version is used (preferably one made for your loader on another Minecraft version), with a warning telling what it was made for. The same goes for its dependencies. `packman update` stays strict: a forced entry is left as it is until a compatible version exists.

### 🔄 Update

`packman update` moves everything to its latest version compatible with the modpack, and adds the required dependencies that are missing. Use `--dry-run` to see what would change.

To migrate the modpack to another Minecraft version or loader, change `mc_version`, `loader` and `loader_version` in `packman.json`, then run `packman update`. Entries with no compatible version are listed and left as they are.

### 📤 Export

`packman export` generates `dist/<name>-<version>.mrpack` and `dist/<name>-<version>.zip`.

- Content available on the target platform is referenced there.
- Content available on the other platform only is downloaded and bundled in the pack as an override. Bundling a CurseForge-only file needs your API key, and is impossible if its author disallows third-party distribution (use `--skip-missing` to leave it out).
- Bundled files are redistributed inside the pack: make sure their licenses allow it.

### 📋 Content list

`packman list` prints the content of the modpack as a Markdown list, one section per type, sorted by name:

```md
# List of projects used

- [Mods used](#mods-used)

## Mods used

- [Sodium](https://modrinth.com/project/AANobbMI)
```

- `--platform modrinth|curseforge` chooses where the links point to (default: `modrinth`). Content that isn't on that platform links to the other one.
- CurseForge page URLs need your API key. Without it, links use the project ID (`curseforge.com/projects/<id>`), which redirects to the page.
- `-o PACK_CONTENT.md` writes the list to a file, and `--type` restricts it to mods, resource packs or shaders.

## ❌ Uninstallation

### 👤 For users

```sh
rm $(go env GOPATH)/bin/packman
```

Delete the saved CurseForge API key first if you don't want to keep it:

```sh
packman auth curseforge --remove
```

### 👨‍💻 For devs

You just need to remove the `packman` directory

## 🤝 Contributing

packman is a work in progress, and all suggestions are welcome.

If you'd like to contribute:

- Open an issue to suggest an idea, report a bug, or discuss a change
- Submit a pull request if you have an improvement to propose

## To Do

- Remove the dependencies that are no longer needed
- Warn about incompatible mods
- Documentation (website)
- Create a TUI
