#!/bin/bash

set -e

UNSTRIPPED_BINARY="dummy_ssl_unstripped"

if [ ! -f "$UNSTRIPPED_BINARY" ]; then
    echo "Error: Unstripped binary '$UNSTRIPPED_BINARY' not found. Run compile.sh first."
    exit 1
fi

echo "Finding offsets in $UNSTRIPPED_BINARY..."

# Get the initial offsets (which point to endbr64)
OFFSET_READ_RAW=$(nm ${UNSTRIPPED_BINARY} | grep ' T dummy_SSL_read$' | awk '{print $1}')
OFFSET_WRITE_RAW=$(nm ${UNSTRIPPED_BINARY} | grep ' T dummy_SSL_write$' | awk '{print $1}')

if [ -z "$OFFSET_READ_RAW" ] || [ -z "$OFFSET_WRITE_RAW" ]; then
    echo "Error: Could not find one or both function offsets."
    echo "Output from nm:"
    nm ${UNSTRIPPED_BINARY} | grep ' dummy_SSL_'
    exit 1
fi

# Add 4 bytes to skip the endbr64 instruction
OFFSET_READ=$(printf "0x%x" $((0x${OFFSET_READ_RAW} + 4)))
OFFSET_WRITE=$(printf "0x%x" $((0x${OFFSET_WRITE_RAW} + 4)))


echo "Raw offset for dummy_SSL_read:  0x${OFFSET_READ_RAW}"
echo "Raw offset for dummy_SSL_write: 0x${OFFSET_WRITE_RAW}"
echo "Adjusted offset for dummy_SSL_read (+4): ${OFFSET_READ}"
echo "Adjusted offset for dummy_SSL_write (+4): ${OFFSET_WRITE}"

export DUMMY_READ_OFFSET=${OFFSET_READ}
export DUMMY_WRITE_OFFSET=${OFFSET_WRITE}
echo "Adjusted offsets exported as DUMMY_READ_OFFSET and DUMMY_WRITE_OFFSET"
