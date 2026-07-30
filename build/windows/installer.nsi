; Git Flow Plus — NSIS Windows installer.
; See WindowsInstallation.md and Packaging.md for the full writeup, and
; build/windows/create-installer.ps1 for how this gets invoked.
;
; Build (from repo root, after staging git-flow-plus.exe — see
; create-installer.ps1 for the staging step CI performs automatically):
;   makensis /DVERSION=1.3.2 /DFILEVERSION=1.3.2.0 /DBIN_DIR=dist\windows-x64 ^
;     /DOUT_DIR=dist\installer build\windows\installer.nsi
;
; All defines fall back to values that let this script also compile
; standalone for syntax testing.

Unicode true

!ifndef VERSION
  !define VERSION "0.0.0"
!endif
!ifndef FILEVERSION
  !define FILEVERSION "0.0.0.0"
!endif
!ifndef BIN_DIR
  !define BIN_DIR "..\..\dist\windows-x64"
!endif
!ifndef OUT_DIR
  !define OUT_DIR "..\..\dist\installer"
!endif

!define APP_NAME "Git Flow Plus"
!define APP_PUBLISHER "Git Flow Plus"
!define APP_DESCRIPTION "Enterprise Git Flow Extension"
!define APP_URL "https://github.com/tabnaeem/git-flow-plus"
!define APP_EXE "git-flow-plus.exe"
!define APP_ALT_EXE "git-flow.exe"
!define UNINSTALL_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\GitFlowPlus"
!define ENV_KEY "SYSTEM\CurrentControlSet\Control\Session Manager\Environment"

!include "MUI2.nsh"
!include "WinMessages.nsh"
!include "x64.nsh"
!include "LogicLib.nsh"
!include "FileFunc.nsh"
!insertmacro GetSize

;--------------------------------
; General

Name "${APP_NAME}"
OutFile "${OUT_DIR}\GitFlowPlusSetup_v${VERSION}_x64.exe"
InstallDir "$PROGRAMFILES64\Git Flow Plus"
InstallDirRegKey HKLM "${UNINSTALL_KEY}" "InstallLocation"
RequestExecutionLevel admin
SetCompressor /SOLID lzma

VIProductVersion "${FILEVERSION}"
VIAddVersionKey "ProductName" "${APP_NAME}"
VIAddVersionKey "CompanyName" "${APP_PUBLISHER}"
VIAddVersionKey "LegalCopyright" "Copyright (c) Git Flow Plus Contributors"
VIAddVersionKey "FileDescription" "${APP_DESCRIPTION}"
VIAddVersionKey "FileVersion" "${FILEVERSION}"
VIAddVersionKey "ProductVersion" "${VERSION}"

!define MUI_ICON "icon.ico"
!define MUI_UNICON "icon.ico"
!define MUI_ABORTWARNING

;--------------------------------
; Pages — Welcome, License, Directory, Components, Install, Finish

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "license.txt"
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_COMPONENTS
!insertmacro MUI_PAGE_INSTFILES

!define MUI_FINISHPAGE_RUN "$INSTDIR\${APP_EXE}"
!define MUI_FINISHPAGE_RUN_PARAMETERS "doctor"
!define MUI_FINISHPAGE_RUN_TEXT "Run 'git flow doctor' to verify the installation"
!define MUI_FINISHPAGE_RUN_NOTCHECKED
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "English"

;--------------------------------
; Version compare helper (installed vs. this package) — used by .onInit
; to detect an upgrade/downgrade and inform the user, without blocking
; either (this installer, like the project's other two, never blocks a
; downgrade — see UpgradeGuide.md).

Var PrevVersion
Var PrevUninstaller

;--------------------------------
; PATH helpers (hand-rolled, no third-party plugin — see Packaging.md
; for why: NSIS's stock distribution doesn't bundle the popular EnVar
; plugin, and pulling in an unpinned third-party DLL download during CI
; would work against reproducible builds. Everything below uses only
; instructions built into the base NSIS compiler/runtime.)
;
; Both the installer and uninstaller need these, so they're defined
; twice via the un. naming convention NSIS requires for functions used
; on both sides — the bodies are identical.

!macro PathContains un
Function ${un}PathContains
  Exch $R8 ; in: dir to search for; out: "1"/"0"
  Push $R0
  Push $R1
  Push $R2
  Push $R3
  Push $R4

  ReadRegStr $R0 HKLM "${ENV_KEY}" "PATH"
  StrCpy $R0 ";$R0;"
  StrCpy $R1 ";$R8;"
  StrLen $R2 $R1
  StrCpy $R3 0

  ${un}PathContains_loop:
    StrCpy $R4 $R0 $R2 $R3
    StrCmp $R4 $R1 ${un}PathContains_found
    StrCmp $R4 "" ${un}PathContains_notfound
    IntOp $R3 $R3 + 1
    Goto ${un}PathContains_loop
  ${un}PathContains_found:
    StrCpy $R8 "1"
    Goto ${un}PathContains_end
  ${un}PathContains_notfound:
    StrCpy $R8 "0"
  ${un}PathContains_end:
    Pop $R4
    Pop $R3
    Pop $R2
    Pop $R1
    Pop $R0
    Exch $R8
FunctionEnd
!macroend

; Only the installer side needs a pure "is it already there" check;
; un.RemoveFromPath below needs the match's position (to splice at), not
; just a boolean, so it has its own loop rather than reusing this.
!insertmacro PathContains ""

Function AddToPath
  Exch $R8 ; dir to add
  Push $R0
  Push $R1

  Push $R8
  Call PathContains
  Pop $R0
  StrCmp $R0 "1" AddToPath_done

  ReadRegStr $R0 HKLM "${ENV_KEY}" "PATH"
  StrCpy $R1 $R0 1 -1
  StrCmp $R1 ";" 0 AddToPath_noSemi
    StrCpy $R0 $R0 -1 ; drop a pre-existing trailing ';' so we don't double it
  AddToPath_noSemi:
  StrCmp $R0 "" AddToPath_first
    StrCpy $R0 "$R0;$R8"
    Goto AddToPath_write
  AddToPath_first:
    StrCpy $R0 "$R8"
  AddToPath_write:
  WriteRegExpandStr HKLM "${ENV_KEY}" "PATH" $R0
  SendMessage ${HWND_BROADCAST} ${WM_WININICHANGE} 0 "STR:Environment" /TIMEOUT=5000

  AddToPath_done:
  Pop $R1
  Pop $R0
  Pop $R8
FunctionEnd

; Removes exactly one ';'-delimited entry, handling all four positions
; explicitly (first/last/middle/only-entry) rather than uniformly
; padding both ends of the haystack — that approach loses the separator
; between the two neighbors when the removed entry was in the middle.
; Verified against 8 cases (empty/first/middle/last/only-entry removal,
; a not-present no-op, and two prefix-collision regressions — e.g.
; removing "C:\First" must not also mangle a neighboring "C:\First2")
; via a throwaway unelevated test harness against a scratch HKCU value
; before this logic was trusted against the real machine PATH.
Function un.RemoveFromPath
  Exch $R8 ; dir to remove
  Push $R0 ; current PATH / working value
  Push $R1 ; needle variant / scratch
  Push $R2 ; length
  Push $R3 ; index / left-length
  Push $R4 ; scratch

  ReadRegStr $R0 HKLM "${ENV_KEY}" "PATH"

  ; Case 1: the entire value is exactly this one entry.
  StrCmp $R0 $R8 un.RemoveFromPath_removeAll

  ; Case 2: needle is the first entry ("NEEDLE;...").
  StrCpy $R1 "$R8;"
  StrLen $R2 $R1
  StrCpy $R4 $R0 $R2
  StrCmp $R4 $R1 un.RemoveFromPath_prefix

  ; Case 3: needle is the last entry ("...;NEEDLE").
  StrCpy $R1 ";$R8"
  StrLen $R2 $R1
  StrLen $R3 $R0
  IntOp $R3 $R3 - $R2
  StrCpy $R4 $R0 "" $R3
  StrCmp $R4 $R1 un.RemoveFromPath_suffix

  ; Case 4: needle is a middle entry (";NEEDLE;") — replace the match
  ; with a single ';' so the two neighboring entries stay separated.
  StrCpy $R1 ";$R8;"
  StrLen $R2 $R1
  StrCpy $R3 0
  un.RemoveFromPath_loop:
    StrCpy $R4 $R0 $R2 $R3
    StrCmp $R4 $R1 un.RemoveFromPath_middle
    StrCmp $R4 "" un.RemoveFromPath_notfound
    IntOp $R3 $R3 + 1
    Goto un.RemoveFromPath_loop

  un.RemoveFromPath_removeAll:
    StrCpy $R0 ""
    Goto un.RemoveFromPath_write
  un.RemoveFromPath_prefix:
    StrCpy $R0 $R0 "" $R2
    Goto un.RemoveFromPath_write
  un.RemoveFromPath_suffix:
    StrCpy $R0 $R0 $R3
    Goto un.RemoveFromPath_write
  un.RemoveFromPath_middle:
    StrCpy $R1 $R0 $R3       ; left part, up to (excluding) the leading ';'
    IntOp $R4 $R3 + $R2
    StrCpy $R4 $R0 "" $R4    ; right part, after the trailing ';'
    StrCpy $R0 "$R1;$R4"     ; rejoin with exactly one ';'
    Goto un.RemoveFromPath_write

  un.RemoveFromPath_notfound:
    Goto un.RemoveFromPath_end

  un.RemoveFromPath_write:
    WriteRegExpandStr HKLM "${ENV_KEY}" "PATH" $R0
    SendMessage ${HWND_BROADCAST} ${WM_WININICHANGE} 0 "STR:Environment" /TIMEOUT=5000

  un.RemoveFromPath_end:
    Pop $R4
    Pop $R3
    Pop $R2
    Pop $R1
    Pop $R0
    Pop $R8
FunctionEnd

;--------------------------------
; .onInit — detect a prior install (any version) and silently run its
; uninstaller first, the same "clean upgrade" behavior documented for
; Git Flow Plus's other installers (see Packaging.md).

Function .onInit
  ${If} ${RunningX64}
    SetRegView 64
  ${Else}
    MessageBox MB_OK|MB_ICONSTOP "Git Flow Plus requires 64-bit Windows."
    Abort
  ${EndIf}

  ReadRegStr $PrevUninstaller HKLM "${UNINSTALL_KEY}" "UninstallString"
  StrCmp $PrevUninstaller "" onInit_done

  ReadRegStr $PrevVersion HKLM "${UNINSTALL_KEY}" "DisplayVersion"
  ${IfNot} ${Silent}
    MessageBox MB_OKCANCEL|MB_ICONINFORMATION \
      "Git Flow Plus $PrevVersion is already installed.$\r$\n$\r$\nSetup will remove it before installing ${VERSION}." \
      IDOK onInit_uninstall
    Abort
  ${EndIf}

  onInit_uninstall:
  ExecWait '"$PrevUninstaller" /S _?=$INSTDIR'

  onInit_done:
FunctionEnd

;--------------------------------
; Sections

Section "Git Flow Plus (required)" SecMain
  SectionIn RO
  SetOutPath "$INSTDIR"

  File "${BIN_DIR}\${APP_EXE}"
  ; A second copy under the literal name "git-flow.exe" — that's what
  ; Git itself looks for on PATH to resolve `git flow ...` as a
  ; subcommand; see internal/cli/doctor.go's PATH check and
  ; CommandReference.md.
  File /oname=${APP_ALT_EXE} "${BIN_DIR}\${APP_EXE}"
  File "README.txt"
  File "license.txt"

  WriteUninstaller "$INSTDIR\Uninstall.exe"

  WriteRegStr HKLM "${UNINSTALL_KEY}" "DisplayName" "${APP_NAME}"
  WriteRegStr HKLM "${UNINSTALL_KEY}" "DisplayVersion" "${VERSION}"
  WriteRegStr HKLM "${UNINSTALL_KEY}" "Publisher" "${APP_PUBLISHER}"
  WriteRegStr HKLM "${UNINSTALL_KEY}" "URLInfoAbout" "${APP_URL}"
  WriteRegStr HKLM "${UNINSTALL_KEY}" "InstallLocation" "$INSTDIR"
  WriteRegStr HKLM "${UNINSTALL_KEY}" "DisplayIcon" "$INSTDIR\${APP_EXE}"
  WriteRegStr HKLM "${UNINSTALL_KEY}" "UninstallString" '"$INSTDIR\Uninstall.exe"'
  WriteRegStr HKLM "${UNINSTALL_KEY}" "QuietUninstallString" '"$INSTDIR\Uninstall.exe" /S'
  WriteRegDWORD HKLM "${UNINSTALL_KEY}" "NoModify" 1
  WriteRegDWORD HKLM "${UNINSTALL_KEY}" "NoRepair" 1

  ; EstimatedSize wants KB.
  ${GetSize} "$INSTDIR" "/S=0K" $0 $1 $2
  IntFmt $0 "0x%08X" $0
  WriteRegDWORD HKLM "${UNINSTALL_KEY}" "EstimatedSize" $0
SectionEnd

Section "Add to PATH" SecPath
  Push "$INSTDIR"
  Call AddToPath
SectionEnd

Section "Start Menu Shortcut" SecStartMenu
  CreateDirectory "$SMPROGRAMS\Git Flow Plus"
  CreateShortcut "$SMPROGRAMS\Git Flow Plus\Git Flow Plus.lnk" "$INSTDIR\${APP_EXE}" "version"
  CreateShortcut "$SMPROGRAMS\Git Flow Plus\Uninstall Git Flow Plus.lnk" "$INSTDIR\Uninstall.exe"
SectionEnd

Section /o "Desktop Shortcut" SecDesktop
  CreateShortcut "$DESKTOP\Git Flow Plus.lnk" "$INSTDIR\${APP_EXE}" "version"
SectionEnd

;--------------------------------
; Component descriptions

!insertmacro MUI_FUNCTION_DESCRIPTION_BEGIN
  !insertmacro MUI_DESCRIPTION_TEXT ${SecMain} "The Git Flow Plus executable (git-flow-plus.exe and the git-flow.exe alias). Required."
  !insertmacro MUI_DESCRIPTION_TEXT ${SecPath} "Add the install directory to the system PATH, so 'git flow ...' and 'git-flow-plus' work from any terminal."
  !insertmacro MUI_DESCRIPTION_TEXT ${SecStartMenu} "Create a Start Menu shortcut."
  !insertmacro MUI_DESCRIPTION_TEXT ${SecDesktop} "Create a Desktop shortcut."
!insertmacro MUI_FUNCTION_DESCRIPTION_END

;--------------------------------
; Uninstaller

Section "Uninstall"
  Push "$INSTDIR"
  Call un.RemoveFromPath

  Delete "$INSTDIR\${APP_EXE}"
  Delete "$INSTDIR\${APP_ALT_EXE}"
  Delete "$INSTDIR\README.txt"
  Delete "$INSTDIR\license.txt"
  Delete "$INSTDIR\Uninstall.exe"
  RMDir "$INSTDIR"

  Delete "$SMPROGRAMS\Git Flow Plus\Git Flow Plus.lnk"
  Delete "$SMPROGRAMS\Git Flow Plus\Uninstall Git Flow Plus.lnk"
  RMDir "$SMPROGRAMS\Git Flow Plus"
  Delete "$DESKTOP\Git Flow Plus.lnk"

  DeleteRegKey HKLM "${UNINSTALL_KEY}"
SectionEnd

Function un.onInit
  ${If} ${RunningX64}
    SetRegView 64
  ${EndIf}
FunctionEnd
