@echo off
setlocal EnableExtensions

rem Instala o Duck no Windows usando apenas CMD (sem PowerShell).
rem Uso: install-windows.cmd [pasta_destino]

set "REPO=IKauedev/duck"
set "ARCH=amd64"
if /I "%PROCESSOR_ARCHITECTURE%"=="ARM64" set "ARCH=arm64"
if /I "%PROCESSOR_ARCHITEW6432%"=="ARM64" set "ARCH=arm64"

set "INSTALL_DIR=%USERPROFILE%\bin"
if not "%~1"=="" set "INSTALL_DIR=%~1"

set "WORKDIR=%TEMP%\duck-install"
set "ARCHIVE=%WORKDIR%\duck_windows_%ARCH%.zip"
set "URL=https://github.com/%REPO%/releases/latest/download/duck_windows_%ARCH%.zip"

echo Duck - instalacao via CMD
echo Destino: %INSTALL_DIR%
echo URL: %URL%
echo.

if not exist "%WORKDIR%" mkdir "%WORKDIR%"

echo Baixando release...
where curl >nul 2>&1
if %ERRORLEVEL%==0 (
  curl -fsSL -o "%ARCHIVE%" "%URL%"
  if %ERRORLEVEL%==0 goto extract
)

certutil -urlcache -split -f "%URL%" "%ARCHIVE%" >nul
if %ERRORLEVEL%==0 goto extract

where bitsadmin >nul 2>&1
if %ERRORLEVEL%==0 (
  bitsadmin /transfer duckDownload /download /priority normal "%URL%" "%ARCHIVE%" >nul
  if %ERRORLEVEL%==0 goto extract
)

echo Erro: nao foi possivel baixar o Duck. Tente baixar manualmente:
echo %URL%
exit /b 1

:extract
echo Extraindo...
tar -xf "%ARCHIVE%" -C "%WORKDIR%"
if %ERRORLEVEL% NEQ 0 (
  echo Erro ao extrair o zip. Confirme que o comando tar esta disponivel no Windows 10+.
  exit /b 1
)

if not exist "%INSTALL_DIR%" mkdir "%INSTALL_DIR%"
copy /Y "%WORKDIR%\duck.exe" "%INSTALL_DIR%\duck.exe" >nul
if %ERRORLEVEL% NEQ 0 (
  echo Erro ao copiar duck.exe para %INSTALL_DIR%
  exit /b 1
)

echo Instalando PATH do usuario...
"%INSTALL_DIR%\duck.exe" install --dir "%INSTALL_DIR%" --force
if %ERRORLEVEL% NEQ 0 (
  echo Duck copiado, mas falhou ao configurar PATH automaticamente.
  echo Adicione manualmente ao PATH: %INSTALL_DIR%
  exit /b 1
)

echo.
echo Duck instalado com sucesso.
echo Abra um novo CMD e execute: duck help
exit /b 0
