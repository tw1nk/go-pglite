#go-pglite uses a wasi version of pglite to have an embeddable postgres database in go

* Currently only a extremely limited REPL somewhat works. No data is returned from the postgres wasi binary


This is based on: github.com/sgosiaco/pglite-go which in turn is inspired by this comment: https://github.com/electric-sql/pglite/issues/89#issuecomment-2418437346
the file: embedfiles/pglite-wasi.tar.gz is from that zip file (unfortunately I have no idea where that came from or how to rebuild pglite with wasi support) 
