packer {
  required_plugins {
    lume = {
      version = ">= v0.0.1"
      # version = "= 0.0.1-dev"
      source = "github.com/warpbuilds/lume"
    }
  }
}

variable "vm_name" {
  type = string
}

variable "vm_base_name" {
  type    = string
  default = ""
}

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

# variable "ipsw" {
#   type    = string
#   default = ""
# }

source "lume-cli" "lume" {
  # ipsw         = var.ipsw
  vm_base_name = var.vm_base_name
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
    "<wait60s>${var.vm_password}<enter>",
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
}