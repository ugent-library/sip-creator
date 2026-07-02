#!/usr/bin/env bash

rm -rf basic-uuid
go run main.go create --profile basic ./tmp/basic basic-uuid
cd basic-uuid
# zip -r test.zip uuid-*
ls | grep uuid | xargs -n 1 -I FOO csip validate --inputs=FOO
catmandu convert JSON to CSV --fix "copy_field(testing.outcome,outcome);copy_field(testing.notes,notes)" < <(jq .validation *.json| less) --fields outcome,id,level,notes | grep "FAILED"