#!/bin/bash

set -e

UNSTRIPPED_BINARY="dummy_ssl_unstripped"

if [ ! -f "$UNSTRIPPED_BINARY" ]; then
    echo "Error: Unstripped binary '$UNSTRIPPED_BINARY' not found. Run compile.sh first."
    exit 1
fi

echo "Finding offsets in $UNSTRIPPED_BINARY..."

OFFSET_READ=$(nm ${UNSTRIPPED_BINARY} | grep ' T dummy_SSL_read$' | awk '{print "0x"$1}')
OFFSET_WRITE=$(nm ${UNSTRIPPED_BINARY} | grep ' T dummy_SSL_write$' | awk '{print "0x"$1}')

if [ -z "$OFFSET_READ" ] || [ -z "$OFFSET_WRITE" ]; then
    echo "Error: Could not find one or both function offsets."
    echo "Output from nm:"
    nm ${UNSTRIPPED_BINARY} | grep ' dummy_SSL_'
    exit 1
fi

echo "Offset for dummy_SSL_read:  ${OFFSET_READ}"
echo "Offset for dummy_SSL_write: ${OFFSET_WRITE}"

export DUMMY_READ_OFFSET=${OFFSET_READ}
export DUMMY_WRITE_OFFSET=${OFFSET_WRITE}
echo "Offsets exported as DUMMY_READ_OFFSET and DUMMY_WRITE_OFFSET"
