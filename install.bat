@echo off
setlocal

cd /d "%~dp0"

where go >nul 2>nul
if errorlevel 1 (
    echo Go nao foi encontrado no PATH.
    echo Instale em https://go.dev/dl/ e rode este instalador de novo.
    exit /b 1
)

for /f "delims=" %%i in ('go env GOBIN') do set "GTR_BIN_DIR=%%i"
if "%GTR_BIN_DIR%"=="" (
    for /f "delims=" %%i in ('go env GOPATH') do set "GTR_BIN_DIR=%%i\bin"
)

set CGO_ENABLED=0
go build -o "%GTR_BIN_DIR%\gtr.exe" .
if errorlevel 1 (
    echo.
    echo Build falhou - veja o erro do go build acima.
    exit /b 1
)

echo.
echo gtr instalado em %GTR_BIN_DIR%\gtr.exe

echo ";%PATH%;" | find /i ";%GTR_BIN_DIR%;" >nul
if errorlevel 1 (
    echo.
    echo Aviso: %GTR_BIN_DIR% nao esta no seu PATH.
    echo Adicione essa pasta ao PATH para rodar "gtr" de qualquer lugar.
) else (
    echo Rode "gtr --help" de qualquer lugar para conferir.
)

endlocal
