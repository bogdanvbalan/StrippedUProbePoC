#!/bin/bash

set -e

UNSTRIPPED_BINARY="dummy_ssl_unstripped"

if [ ! -f "$UNSTRIPPED_BINARY" ]; then
    echo "Error: Unstripped binary '$UNSTRIPPED_BINARY' not found. Run compile.sh first."
    exit 1
fi

echo "Finding offsets in $UNSTRIPPED_BINARY..."

# Get the absolute function VMAs
VMA_READ_RAW=$(nm ${UNSTRIPPED_BINARY} | grep ' T dummy_SSL_read$' | awk '{print $1}')
VMA_WRITE_RAW=$(nm ${UNSTRIPPED_BINARY} | grep ' T dummy_SSL_write$' | awk '{print $1}')

if [ -z "$VMA_READ_RAW" ] || [ -z "$VMA_WRITE_RAW" ]; then
    echo "Error: Could not find one or both function VMAs."
    echo "Output from nm:"
    nm ${UNSTRIPPED_BINARY} | grep ' dummy_SSL_'
    exit 1
fi

# Get the base VMA of the .text section
# Use readelf -S, find the .text line, get the Addr
BASE_VMA_TEXT=$(readelf -S ${UNSTRIPPED_BINARY} | grep ' .text' | awk '{print $4}')

if [ -z "$BASE_VMA_TEXT" ]; then
    echo "Error: Could not find .text section base VMA using readelf."
    readelf -S ${UNSTRIPPED_BINARY}
    exit 1
fi

echo "Base VMA for .text section: 0x${BASE_VMA_TEXT}"
echo "Absolute VMA for dummy_SSL_read:  0x${VMA_READ_RAW}"
echo "Absolute VMA for dummy_SSL_write: 0x${VMA_WRITE_RAW}"

# Calculate relative offsets
OFFSET_READ_RELATIVE=$((0x${VMA_READ_RAW} - 0x${BASE_VMA_TEXT}))
OFFSET_WRITE_RELATIVE=$((0x${VMA_WRITE_RAW} - 0x${BASE_VMA_TEXT}))

# Add 4 bytes to skip the endbr64 instruction
ENDBR_ADJUSTMENT=4
OFFSET_READ_ADJUSTED=$((${OFFSET_READ_RELATIVE} + ${ENDBR_ADJUSTMENT}))
OFFSET_WRITE_ADJUSTED=$((${OFFSET_WRITE_RELATIVE} + ${ENDBR_ADJUSTMENT}))

# Convert final offsets back to hex for clarity/consistency 
OFFSET_READ_HEX=$(printf "0x%x" ${OFFSET_READ_ADJUSTED})
OFFSET_WRITE_HEX=$(printf "0x%x" ${OFFSET_WRITE_ADJUSTED})

echo "Relative offset for dummy_SSL_read:  0x$(printf %x ${OFFSET_READ_RELATIVE})"
echo "Relative offset for dummy_SSL_write: 0x$(printf %x ${OFFSET_WRITE_RELATIVE})"
echo "Adjusted offset for dummy_SSL_read (+${ENDBR_ADJUSTMENT}): ${OFFSET_READ_HEX}"
echo "Adjusted offset for dummy_SSL_write (+${ENDBR_ADJUSTMENT}): ${OFFSET_WRITE_HEX}"

# Export the *final adjusted relative offsets*
export DUMMY_READ_OFFSET=${OFFSET_READ_HEX}
export DUMMY_WRITE_OFFSET=${OFFSET_WRITE_HEX}
echo "Adjusted relative offsets exported as DUMMY_READ_OFFSET and DUMMY_WRITE_OFFSET"