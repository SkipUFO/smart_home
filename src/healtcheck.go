package main

/*
https://github.com/jackc/pgx/issues/891

Your examples seem to be mixing a closed connection vs. a closed pool... But you can try to acquire a connection as a test of the pool. If you get the closed pool error then the pool is closed.
*/
