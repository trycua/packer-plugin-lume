#!/bin/bash

# Define the directory to search
LUME_DIR="$HOME/.lume"

# Check if the directory exists
if [ ! -d "$LUME_DIR" ]; then
  echo "Directory $LUME_DIR does not exist."
  exit 1
fi

# Use a regex pattern to match UUID-like directory names
UUID_PATTERN='^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$'

# Find directories with UUID names
for dir in "$LUME_DIR"/*; do
  if [ -d "$dir" ]; then
    dir_name=$(basename "$dir")
    if [[ $dir_name =~ $UUID_PATTERN ]]; then
      echo "Deleting: $dir"
      rm -rf $dir
    fi
  fi
done

vm_name=v002
lume delete --force v002
echo "Deleting vm $vm_name"
rm -rf $vm_name