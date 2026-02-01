; OpenPrintHub Windows Installer Script
; Requires NSIS 3.x

!include "MUI2.nsh"
!include "FileFunc.nsh"

; --------------------------------
; General
; --------------------------------
Name "OpenPrintHub"
OutFile "OpenPrintHub-Setup.exe"
InstallDir "$PROGRAMFILES64\OpenPrintHub"
InstallDirRegKey HKLM "Software\OpenPrintHub" "InstallDir"
RequestExecutionLevel admin

; --------------------------------
; Version Info
; --------------------------------
!ifndef VERSION
  !define VERSION "0.1.0"
!endif

VIProductVersion "${VERSION}.0"
VIAddVersionKey "ProductName" "OpenPrintHub"
VIAddVersionKey "CompanyName" "OpenPrintHub"
VIAddVersionKey "LegalCopyright" "MIT License"
VIAddVersionKey "FileDescription" "OpenPrintHub Silent Printing Service"
VIAddVersionKey "FileVersion" "${VERSION}"
VIAddVersionKey "ProductVersion" "${VERSION}"

; --------------------------------
; Interface Settings
; --------------------------------
!define MUI_ABORTWARNING
!define MUI_ICON "${NSISDIR}\Contrib\Graphics\Icons\modern-install.ico"
!define MUI_UNICON "${NSISDIR}\Contrib\Graphics\Icons\modern-uninstall.ico"

; --------------------------------
; Pages
; --------------------------------
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "..\..\LICENSE"
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

; --------------------------------
; Languages
; --------------------------------
!insertmacro MUI_LANGUAGE "English"
!insertmacro MUI_LANGUAGE "SimpChinese"

; --------------------------------
; Installer Section
; --------------------------------
Section "Install"
  SetOutPath "$INSTDIR"
  
  ; Kill any running instance
  nsExec::ExecToLog 'taskkill /F /IM oph.exe'
  Sleep 1000
  
  ; Copy files
  File "oph.exe"
  File "..\..\LICENSE"
  File "..\..\README.md"
  
  ; Create start menu shortcuts
  CreateDirectory "$SMPROGRAMS\OpenPrintHub"
  CreateShortcut "$SMPROGRAMS\OpenPrintHub\OpenPrintHub.lnk" "$INSTDIR\oph.exe"
  CreateShortcut "$SMPROGRAMS\OpenPrintHub\Uninstall.lnk" "$INSTDIR\uninstall.exe"
  
  ; Create desktop shortcut
  CreateShortcut "$DESKTOP\OpenPrintHub.lnk" "$INSTDIR\oph.exe"
  
  ; Add to Windows startup (auto-start on login)
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Run" \
                   "OpenPrintHub" "$INSTDIR\oph.exe"
  
  ; Write uninstaller
  WriteUninstaller "$INSTDIR\uninstall.exe"
  
  ; Write registry keys
  WriteRegStr HKLM "Software\OpenPrintHub" "InstallDir" "$INSTDIR"
  WriteRegStr HKLM "Software\OpenPrintHub" "Version" "${VERSION}"
  
  ; Add to Add/Remove Programs
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\OpenPrintHub" \
                   "DisplayName" "OpenPrintHub"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\OpenPrintHub" \
                   "UninstallString" "$\"$INSTDIR\uninstall.exe$\""
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\OpenPrintHub" \
                   "InstallLocation" "$INSTDIR"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\OpenPrintHub" \
                   "DisplayVersion" "${VERSION}"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\OpenPrintHub" \
                   "Publisher" "OpenPrintHub"
  WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\OpenPrintHub" \
                     "NoModify" 1
  WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\OpenPrintHub" \
                     "NoRepair" 1
  
  ; Get installed size
  ${GetSize} "$INSTDIR" "/S=0K" $0 $1 $2
  IntFmt $0 "0x%08X" $0
  WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\OpenPrintHub" \
                     "EstimatedSize" "$0"
  
  ; Start the application
  Exec '"$INSTDIR\oph.exe"'
  
SectionEnd

; --------------------------------
; Uninstaller Section
; --------------------------------
Section "Uninstall"
  ; Kill running instance
  nsExec::ExecToLog 'taskkill /F /IM oph.exe'
  Sleep 1000
  
  ; Remove from startup
  DeleteRegValue HKCU "Software\Microsoft\Windows\CurrentVersion\Run" "OpenPrintHub"
  
  ; Remove files
  Delete "$INSTDIR\oph.exe"
  Delete "$INSTDIR\LICENSE"
  Delete "$INSTDIR\README.md"
  Delete "$INSTDIR\uninstall.exe"
  
  ; Remove shortcuts
  Delete "$SMPROGRAMS\OpenPrintHub\OpenPrintHub.lnk"
  Delete "$SMPROGRAMS\OpenPrintHub\Uninstall.lnk"
  RMDir "$SMPROGRAMS\OpenPrintHub"
  Delete "$DESKTOP\OpenPrintHub.lnk"
  
  ; Remove install directory
  RMDir "$INSTDIR"
  
  ; Remove registry keys
  DeleteRegKey HKLM "Software\OpenPrintHub"
  DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\OpenPrintHub"
  
SectionEnd
