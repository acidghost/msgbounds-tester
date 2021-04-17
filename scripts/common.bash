COVOUT="/work/cov.json"

function check_help {
    local opt
    for opt in "$@"; do
        if [[ "$opt" =~ -h|-help ]]; then
            /work/tester -h
            exit $?
        fi
    done
}

function dump_coverage {
    if ! (gcovr -r "$1" -s | grep branch && gcovr -r "$1" --json > "$COVOUT"); then
        echo "!!! Something went wrong dumping coverage with gcovr"
        exit 1
    fi
}

function check_line {
    local msg=$1 file=$2 ln=$3
    echo "Checking for '$msg'"
    local p=".files[] | select(.file == \"$file\") \
        | .lines[] | select(.line_number == $ln) | .count"
    local res
    if ! res=$(jq "$p" "$COVOUT"); then
        echo "!!! Error running jq"
        return 1
    fi
    if [[ "$res" =~ [0-9]+ ]] && [[ "$res" -gt 0 ]]; then
        echo "$file:$ln covered"
    else
        echo "$file:$ln NOT covered"
    fi
}
