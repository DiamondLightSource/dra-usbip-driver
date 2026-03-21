#!/bin/bash -e

if [ ! -d "${ROOTFS_DIR}" ]; then
	copy_previous
fi

# Make AGENT_VERSION available inside the chroot
if [ -f "${STAGE_DIR}/AGENT_VERSION" ]; then
	install -m 644 "${STAGE_DIR}/AGENT_VERSION" "${ROOTFS_DIR}/tmp/AGENT_VERSION"
fi
