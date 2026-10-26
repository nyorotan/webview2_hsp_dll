set GOOS=windows
set GOARCH=386
set CGO_ENABLED=1
set CC=i686-w64-mingw32-gcc
set CXX=i686-w64-mingw32-g++
go build -buildmode=c-shared -o markdown_hsp.dll ./markdown_dll

@REM  set GOARCH=amd64
@REM  set CGO_ENABLED=1
@REM  set CC=x86_64-w64-mingw32-gcc
@REM  set CXX=x86_64-w64-mingw32-g++
@REM  go build -buildmode=c-shared -o ./x64/markdown_hsp.dll ./markdown_dll
