# Witness Cache

Paul Frazee:

I'm increasingly convinced that many Atmosphere backends start with a local "witness cache" of the repositories.
A witness cache is a copy of the repository records, plus a timestamp of when the record was indexed (the "witness time") which you want to keep

The key feature is: you can replay it

With local replay, you can add new tables or indexes to your backend and quickly backfill the data. If you don't have a witness cache, you would have to do backfill from the network, which is slow

RocksDB or other LSMs are good candidates for a witness cache (good write throughput)

Clickhouse and DuckDB are also good candidates (good compression ratio)

## TODO
