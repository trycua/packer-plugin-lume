packer {
  required_plugins {
    lume = {
      version = ">= v0.0.1"
      # version = "= 0.0.1-dev"
      source = "github.com/trycua/packer-plugin-lume"
    }
  }
}

variable "vm_name" {
  type = string
}

# variable "vm_base_name" {
#   type    = string
#   default = ""
# }

variable "vm_username" {
  type      = string
  default   = "runner"
  sensitive = true
}

variable "vm_password" {
  type      = string
  default   = "runner"
  sensitive = true
}

variable "vcpu_count" {
  type    = number
  default = 6
}

variable "ram_size" {
  type    = string
  default = "8GB"
}

variable "image_os" {
  type    = string
  default = "macos14"
}

variable "ipsw" {
  type    = string
  default = ""
}

source "lume-cli" "lume" {
  ipsw = var.ipsw
  # vm_base_name = var.vm_base_name
  vm_name      = var.vm_name
  cpu_count    = var.vcpu_count
  memory       = var.ram_size
  disk_size    = "40GB"
  ssh_password = var.vm_password
  ssh_username = var.vm_username
  ssh_timeout  = "120s"

  # headless     = true

  boot_command = [
    # hello, hola, bonjour, etc.
    "<wait60s><spacebar>",
    # Language: most of the times we have a list of "English"[1], "English (UK)", etc. with
    # "English" language already selected. If we type "english", it'll cause us to switch
    # to the "English (UK)", which is not what we want. To solve this, we switch to some other
    # language first, e.g. "Italiano" and then switch back to "English". We'll then jump to the
    # first entry in a list of "english"-prefixed items, which will be "English".
    #
    # [1]: should be named "English (US)", but oh well 🤷
    "<wait30s>italiano<esc>english<enter>",
    # Select Your Country and Region
    "<wait30s>united states<leftShiftOn><tab><leftShiftOff><spacebar>",
    # Written and Spoken Languages
    "<wait10s><leftShiftOn><tab><leftShiftOff><spacebar>",
    # Accessibility
    "<wait10s><leftShiftOn><tab><leftShiftOff><spacebar>",
    # Data & Privacy
    "<wait10s><leftShiftOn><tab><leftShiftOff><spacebar>",
    # Migration Assistant
    "<wait10s><tab><tab><tab><spacebar>",
    # Sign In with Your Apple ID
    "<wait10s><leftShiftOn><tab><leftShiftOff><leftShiftOn><tab><leftShiftOff><spacebar>",
    # Are you sure you want to skip signing in with an Apple ID?
    "<wait10s><tab><spacebar>",
    # Terms and Conditions
    "<wait10s><leftShiftOn><tab><leftShiftOff><spacebar>",
    # I have read and agree to the macOS Software License Agreement
    "<wait10s><tab><spacebar>",
    # Create a Computer Account
    "<wait10s>${var.vm_username}<tab><tab>${var.vm_password}<tab>${var.vm_password}<tab><tab><tab><spacebar>",
    # Enable Location Services
    "<wait120s><leftShiftOn><tab><leftShiftOff><spacebar>",
    # Are you sure you don't want to use Location Services?
    "<wait10s><tab><spacebar>",
    # Select Your Time Zone
    "<wait10s><tab>UTC<enter><leftShiftOn><tab><leftShiftOff><spacebar>",
    # Analytics
    "<wait10s><leftShiftOn><tab><leftShiftOff><spacebar>",
    # Screen Time
    "<wait10s><tab><spacebar>",
    # # Siri <- screen seems to be missing in newer ipsw images, commenting for now
    # "<wait10s><tab><spacebar><leftShiftOn><tab><leftShiftOff><spacebar>",
    # Choose Your Look
    "<wait10s><leftShiftOn><tab><leftShiftOff><spacebar>",
    # Welcome to Mac
    "<wait10s><spacebar>",
    # Enable Keyboard navigation
    # This is so that we can navigate the System Settings app using the keyboard
    "<wait10s><leftAltOn><spacebar><leftAltOff>Terminal<enter>",
    "<wait10s>defaults write NSGlobalDomain AppleKeyboardUIMode -int 3<enter>",
    "<wait10s><leftAltOn>q<leftAltOff>",
    # Now that the installation is done, open "System Settings"
    "<wait10s><leftAltOn><spacebar><leftAltOff>System Settings<enter>",
    # Navigate to "Sharing"
    "<wait10s><leftAltOn>f<leftAltOff>sharing<enter>",
    # Navigate to "Screen Sharing" and enable it
    "<wait10s><tab><tab><tab><tab><tab><spacebar>",
    # Navigate to "Remote Login" and enable it
    "<wait10s><tab><tab><tab><tab><tab><tab><tab><tab><tab><tab><tab><tab><spacebar>",
    # Quit System Settings
    "<wait10s><leftAltOn>q<leftAltOff>",
  ]

  # // A (hopefully) temporary workaround for Virtualization.Framework's
  # // installation process not fully finishing in a timely manner
  # create_grace_time = "30s"
}

build {
  sources = ["source.lume-cli.lume"]

  provisioner "shell" {
    environment_vars = ["PASSWORD=${var.vm_password}"]
    scripts          = ["./scripts/prepare/setup-system.sh"]
  }

  post-processor "lume-export" {
    tag = "image"
  }
}