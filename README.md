<div align="center">
<h1>
  <div class="image-wrapper" style="display: inline-block;">
    <picture>
      <source media="(prefers-color-scheme: dark)" alt="logo" height="150" srcset="img/logo_white.png" style="display: block; margin: auto;">
      <source media="(prefers-color-scheme: light)" alt="logo" height="150" srcset="img/logo_black.png" style="display: block; margin: auto;">
      <img alt="Shows my svg">
    </picture>
  </div>

  [![Go](https://img.shields.io/badge/Go-00ADD8?logo=go&logoColor=white&labelColor=00ADD8)](#)
  [![macOS](https://img.shields.io/badge/macOS-000000?logo=apple&logoColor=F0F0F0)](#)
  [![Discord](https://img.shields.io/badge/Discord-%235865F2.svg?&logo=discord&logoColor=white)](https://discord.com/invite/mVnXXpdE85)
</h1>
</div>

**packer-plugin-lume** is a Packer plugin for building macOS and Linux VM images with [Lume](https://github.com/trycua/cua/tree/main/libs/lume) on Apple Silicon. It provides automated VM creation and provisioning through Lume's CLI/API.

## Installation

### Prerequisites

```bash
# Install Go
brew install golang

# Install Packer
brew tap hashicorp/tap
brew install hashicorp/tap/packer
```

### Steps

1. Install Lume if you haven't already:

```bash
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/trycua/cua/main/libs/lume/scripts/install.sh)"
```

2. Obtain a macOS IPSW restore image:
```bash
# Get the latest macOS IPSW download URL
lume ipsw

# Download the IPSW image (this may take a while)
curl -o macOS.ipsw [URL from previous command]
```

3. Build and install the plugin:

```bash
make dev
```

4. Navigate to the `bin` directory and customize `variables.hcl` with your desired configuration and full path to the IPSW image:
```bash
cd bin

# Edit variables.hcl to set vm_name, cpu_count, memory, and path of a IPSW image.
```

## Usage Example

Run the build:

```bash
packer build -var-file=variables.hcl macOS-15.arm64.lume.prepare-01.pkr.hcl
```

## Configuration Reference

### Builder Configuration

| Parameter | Description | Type | Default |
|-----------|-------------|------|---------|
| `vm_name` | Name for the VM | string | Required |
| `vm_base_name` | Base VM to clone | string | Optional |
| `ipsw` | Path to IPSW file or 'latest' | string | Optional |
| `cpu_count` | Number of CPU cores | number | 4 |
| `memory` | Memory size | string | "4GB" |
| `disk_size` | Disk size | string | "40GB" |
| `display` | Display resolution | string | "1024x768" |
| `headless` | Run without display | boolean | false |
| `recovery_mode` | Start in recovery mode | boolean | false |
| `ssh_username` | SSH username | string | Required |
| `ssh_password` | SSH password | string | Required |
| `ssh_timeout` | SSH connection timeout | string | "10m" |

## Contributing

We welcome and greatly appreciate contributions to packer-plugin-lume! Whether you're improving documentation, adding new features, fixing bugs, your efforts help make this packer plugin better for everyone.

Join our [Discord community](https://discord.com/invite/mVnXXpdE85) to discuss ideas or get assistance.

## Acknowledgements

The macOS boot command setup were adapted from the excellent work by [Cirrus Labs](https://github.com/cirruslabs/macos-image-templates), which is licensed under the [MIT License](https://github.com/cirruslabs/macos-image-templates/blob/master/LICENSE). Proper attribution is provided in accordance with the license terms.

Thanks to [PrashantRaj18198](https://github.com/PrashantRaj18198) for starting off the work to port the Tart Packer plugin to Lume.

## License

This project is licensed under the MPL-2.0 License - see the [LICENSE](LICENSE) file for details.

## Trademarks

Apple, macOS, and Apple Silicon are trademarks of Apple Inc. Ubuntu and Canonical are registered trademarks of Canonical Ltd. This project is not affiliated with, endorsed by, or sponsored by Apple Inc. or Canonical Ltd. 
