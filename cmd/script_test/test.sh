#!/bin/bash

function fn_print_help() {
    echo "-m MESSAGE"
    exit ${ERROR_CODE_INVALID}
}

while getopts 'm:' OPT; do
    case $OPT in
        m)
            MESSAGE="$OPTARG";;
        ?)
            fn_print_help
    esac
done

if [ "${MESSAGE}" == "" ]; then
    fn_print_help
fi

echo ${MESSAGE}

exit ${EXIT_CODE_OK}