; Git Flow Plus — Windows Installer (Inno Setup 6).
; See WindowsInstallation.md and Packaging.md for the full writeup of what
; this does and why Inno Setup was chosen over WiX/NSIS.
;
; Build (from repo root), after staging a single flat directory containing
; git-flow-plus.exe for the target architecture — see Packaging.md for the
; staging step; CI does this automatically:
;   iscc /DMyAppVersion=5.3.4.1.2 /DMyAppArch=x64 /DMyAppBinDir=dist\windows-x64 installer\windows\gitflowplus.iss
;
; All three defines fall back to values that let this script also compile
; standalone (e.g. `iscc installer\windows\gitflowplus.iss`) for syntax
; testing, as long as dist\windows-x64\git-flow-plus.exe exists locally.

#ifndef MyAppVersion
  #define MyAppVersion "0.0.0.0.0"
#endif
#ifndef MyAppArch
  #define MyAppArch "x64"
#endif
#ifndef MyAppBinDir
  #define MyAppBinDir "..\..\dist\windows-" + MyAppArch
#endif

; MyAppArch ("x64"/"arm64") is the clean, user-facing label used in
; output filenames below. Inno's own architecture-identifier tokens are
; slightly different (it deprecated bare "x64" in favor of
; "x64compatible", which also covers x64-emulation on Arm64 devices) —
; this maps one to the other so ArchitecturesAllowed/
; ArchitecturesInstallIn64BitMode always use the current, non-deprecated
; token without making filenames ugly.
#if MyAppArch == "arm64"
  #define MyAppArchIdentifier "arm64"
#else
  #define MyAppArchIdentifier "x64compatible"
#endif

#define MyAppName "Git Flow Plus"
#define MyAppPublisher "Git Flow Plus Contributors"
#define MyAppURL "https://github.com/tabnaeem/git-flow-plus"
#define MyAppExeName "git-flow-plus.exe"
#define MyAppAltExeName "git-flow.exe"

; Fixed forever — regenerating this breaks upgrade detection for every
; existing install. Generated once; never regenerate.
;
; Two forms of the same GUID: [Setup]-section values get re-scanned by
; Inno's own {constant} parser after ISPP substitutes them in, so that
; one needs its braces doubled (Inno's literal-brace escape) or it's
; misread as an unknown {constant} reference. Pascal [Code] string
; literals only go through ISPP's textual substitution, so that one
; needs single braces — the real GUID text.
#define MyAppId "{0A0F9E60-2EE6-4D70-9EB7-4B2F643C6C11}"
#define MyAppIdForSetup "{{0A0F9E60-2EE6-4D70-9EB7-4B2F643C6C11}}"

; Win32 VERSIONINFO resources are strictly 4 numeric fields, but Git Flow
; Plus's own version is 5 (Sprint.Feature.ReleaseFix.DevOps.QA, e.g.
; "5.3.4.1.2"). This drops the trailing QA-build digit for the embedded
; file-version resource only; AppVersion above keeps the full string as
; the human-facing display version. See Packaging.md.
#define MyAppFileVersion Copy(MyAppVersion, 1, RPos(".", MyAppVersion) - 1)

[Setup]
AppId={#MyAppIdForSetup}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppVerName={#MyAppName} {#MyAppVersion}
VersionInfoVersion={#MyAppFileVersion}
AppPublisher={#MyAppPublisher}
AppPublisherURL={#MyAppURL}
AppSupportURL={#MyAppURL}/issues
AppUpdatesURL={#MyAppURL}/releases
DefaultDirName={autopf}\GitFlowPlus
DefaultGroupName=Git Flow Plus
DisableProgramGroupPage=yes
PrivilegesRequired=lowest
PrivilegesRequiredOverridesAllowed=commandline dialog
ArchitecturesAllowed={#MyAppArchIdentifier}
ArchitecturesInstallIn64BitMode={#MyAppArchIdentifier}
Compression=lzma2/max
SolidCompression=yes
OutputDir=..\..\dist\installer
OutputBaseFilename=git-flow-plus-{#MyAppVersion}-windows-{#MyAppArch}-setup
UninstallDisplayIcon={app}\{#MyAppExeName}
WizardStyle=modern
ChangesEnvironment=yes
LicenseFile=..\..\LICENSE

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "addtopath"; Description: "Add Git Flow Plus to PATH (required for 'git flow ...' to work)"; GroupDescription: "PATH:"; Flags: checkedonce

[Files]
Source: "{#MyAppBinDir}\{#MyAppExeName}"; DestDir: "{app}"; Flags: ignoreversion
; A second copy under the literal name "git-flow.exe" — that's what Git
; itself looks for on PATH to resolve `git flow ...` as a subcommand; see
; internal/cli/doctor.go's PATH check and CommandReference.md.
Source: "{#MyAppBinDir}\{#MyAppExeName}"; DestDir: "{app}"; DestName: "{#MyAppAltExeName}"; Flags: ignoreversion
Source: "..\..\README.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\..\LICENSE"; DestDir: "{app}"; Flags: ignoreversion
; Bundled but not installed to {app} directly — CreateDefaultConfig below
; extracts and copies it to the seeded config location instead. Shared
; verbatim with the WiX MSI installer, so the default config content
; exists in exactly one place.
Source: "default-config.json"; DestDir: "{tmp}"; Flags: dontcopy

[Icons]
Name: "{group}\{cm:UninstallProgram,{#MyAppName}}"; Filename: "{uninstallexe}"

[Run]
Filename: "{app}\{#MyAppExeName}"; Parameters: "doctor"; Description: "Run 'git flow doctor' to verify the installation"; Flags: postinstall shellexec skipifsilent runasoriginaluser
Filename: "{app}\{#MyAppExeName}"; Parameters: "version"; Description: "Show the installed version"; Flags: postinstall shellexec skipifsilent unchecked runasoriginaluser

[Code]
const
  EnvKeyHKLM = 'SYSTEM\CurrentControlSet\Control\Session Manager\Environment';
  EnvKeyHKCU = 'Environment';
  WM_SETTINGCHANGE = $001A;
  GFP_HWND_BROADCAST = $FFFF;
  SMTO_ABORTIFHUNG = $0002;

{ Declared directly against user32.dll — Inno's Pascal Script supports
  external Win32 API calls natively, which is simpler and more reliable
  here than shelling out to another process just to broadcast one message. }
function SendMessageTimeoutA(hWnd: Integer; Msg: Integer; wParam: Integer; lParam: String;
  fuFlags: Integer; uTimeout: Integer; var lpdwResult: Integer): Integer;
  external 'SendMessageTimeoutA@user32.dll stdcall';

procedure BroadcastEnvironmentChange();
var
  Res: Integer;
begin
  { Lets an already-open Explorer/shell pick up the new PATH without a
    logoff — a freshly opened terminal would see it anyway, but this
    avoids that surprise for anything already running. Best-effort: its
    return value is deliberately ignored. }
  SendMessageTimeoutA(GFP_HWND_BROADCAST, WM_SETTINGCHANGE, 0, 'Environment', SMTO_ABORTIFHUNG, 5000, Res);
end;

procedure EnvRootAndKey(UseHKLM: Boolean; var Root: Integer; var Key: String);
begin
  if UseHKLM then
  begin
    Root := HKLM;
    Key := EnvKeyHKLM;
  end
  else
  begin
    Root := HKCU;
    Key := EnvKeyHKCU;
  end;
end;

{ Appends Dir to the Path value under Root\Key if it isn't already
  present (case-insensitively), so re-running the installer — e.g. during
  an upgrade — never produces duplicate PATH entries. }
function EnvAddPath(UseHKLM: Boolean; Dir: String): Boolean;
var
  Root: Integer;
  Key, OrigPath, NewPath: String;
begin
  EnvRootAndKey(UseHKLM, Root, Key);

  if not RegQueryStringValue(Root, Key, 'Path', OrigPath) then
    OrigPath := '';

  if Pos(';' + Uppercase(Dir) + ';', ';' + Uppercase(OrigPath) + ';') > 0 then
  begin
    Result := True;
    Exit;
  end;

  if (OrigPath <> '') and (OrigPath[Length(OrigPath)] <> ';') then
    NewPath := OrigPath + ';' + Dir
  else
    NewPath := OrigPath + Dir;

  Result := RegWriteExpandStringValue(Root, Key, 'Path', NewPath);
  if Result then
    BroadcastEnvironmentChange();
end;

{ Removes exactly Dir from the Path value under Root\Key, splicing the
  surrounding separators back together — never touches the rest of the
  value. Deliberately does NOT use Inno's [Registry] uninsdeletevalue
  flag for this: that flag deletes the entire Path value on uninstall,
  which would wipe out every other program's PATH entry too. }
function EnvRemovePath(UseHKLM: Boolean; Dir: String): Boolean;
var
  Root: Integer;
  Key, OrigPath, Padded, Needle: String;
  P: Integer;
begin
  EnvRootAndKey(UseHKLM, Root, Key);

  if not RegQueryStringValue(Root, Key, 'Path', OrigPath) then
  begin
    Result := True;
    Exit;
  end;

  Padded := ';' + OrigPath + ';';
  Needle := ';' + Dir + ';';
  P := Pos(Uppercase(Needle), Uppercase(Padded));
  if P > 0 then
    Delete(Padded, P, Length(Needle) - 1);

  if (Length(Padded) > 0) and (Padded[1] = ';') then
    Delete(Padded, 1, 1);
  if (Length(Padded) > 0) and (Padded[Length(Padded)] = ';') then
    Delete(Padded, Length(Padded), 1);

  Result := RegWriteExpandStringValue(Root, Key, 'Path', Padded);
  if Result then
    BroadcastEnvironmentChange();
end;

{ ---------- Detection helpers (informational only; never hard-block install) ---------- }

function DetectGit(): Boolean;
var
  ResultCode: Integer;
begin
  Result := Exec(ExpandConstant('{cmd}'), '/c git --version >nul 2>nul', '', SW_HIDE, ewWaitUntilTerminated, ResultCode)
    and (ResultCode = 0);
end;

function DetectGitBash(): Boolean;
var
  InstallPath: String;
begin
  Result := FileExists(ExpandConstant('{pf}\Git\bin\bash.exe'))
    or FileExists(ExpandConstant('{pf32}\Git\bin\bash.exe'));
  if not Result and RegQueryStringValue(HKLM, 'SOFTWARE\GitForWindows', 'InstallPath', InstallPath) then
    Result := FileExists(AddBackslash(InstallPath) + 'bin\bash.exe');
end;

function DetectPowerShell(): Boolean;
begin
  Result := FileExists(ExpandConstant('{sys}\WindowsPowerShell\v1.0\powershell.exe'));
end;

{ ---------- Upgrade: detect + silently remove a previous install ---------- }

function GetUninstallString(): String;
var
  UninstallKey: String;
begin
  UninstallKey := 'Software\Microsoft\Windows\CurrentVersion\Uninstall\{#MyAppId}_is1';
  Result := '';
  if not RegQueryStringValue(HKLM, UninstallKey, 'UninstallString', Result) then
    RegQueryStringValue(HKCU, UninstallKey, 'UninstallString', Result);
end;

function InitializeSetup(): Boolean;
var
  UninstallString: String;
  ResultCode: Integer;
  Msg: String;
begin
  Result := True;

  { A previous Git Flow Plus install is detected via its own AppId's
    uninstall registry key (the standard Inno "self-upgrade" recipe) and
    removed silently first, so the new version's files never land
    alongside stale ones from an old release. }
  UninstallString := GetUninstallString();
  if UninstallString <> '' then
  begin
    UninstallString := RemoveQuotes(UninstallString);
    Exec(UninstallString, '/VERYSILENT /NORESTART /SUPPRESSMSGBOXES', '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
  end;

  if not WizardSilent() then
  begin
    if not DetectGit() then
    begin
      Msg := 'Git was not detected on PATH.' + #13#10#13#10 +
        'Git Flow Plus requires a working ''git'' installation to function. You can ' + #13#10 +
        'install Git afterwards and run ''git flow doctor'' to verify.' + #13#10#13#10 +
        'Continue installing Git Flow Plus anyway?';
      if MsgBox(Msg, mbConfirmation, MB_YESNO) = IDNO then
      begin
        Result := False;
        Exit;
      end;
    end;
    if not (DetectGitBash() or DetectPowerShell()) then
      MsgBox('Neither Git Bash nor Windows PowerShell was detected.' + #13#10#13#10 +
        'Git Flow Plus works from any shell, but these are the two most commonly ' + #13#10 +
        'used on Windows, so this is worth double-checking your setup for.',
        mbInformation, MB_OK);
  end;
end;

{ ---------- Post-install: PATH + seed %APPDATA%/%LOCALAPPDATA% ---------- }

procedure CreateDefaultConfig();
var
  ConfigDir, LogsDir, ConfigFile: String;
begin
  if IsAdminInstallMode then
  begin
    ConfigDir := ExpandConstant('{commonappdata}\GitFlowPlus');
    LogsDir := ExpandConstant('{commonappdata}\GitFlowPlus\logs');
  end
  else
  begin
    ConfigDir := ExpandConstant('{userappdata}\GitFlowPlus');
    LogsDir := ExpandConstant('{localappdata}\GitFlowPlus\logs');
  end;

  ForceDirectories(ConfigDir);
  { Reserved for future file-based diagnostic logging — Git Flow Plus
    does not yet write here; see Troubleshooting.md. }
  ForceDirectories(LogsDir);

  ConfigFile := ConfigDir + '\config.json';
  if not FileExists(ConfigFile) then
  begin
    { default-config.json mirrors internal/config.Default()'s exact JSON
      shape — a real, valid config a user could copy into a repository's
      .gitflowplus/config.json, not a placeholder. Git Flow Plus itself
      only reads the repo-local file today; see WindowsInstallation.md
      for the current scope of what this seeded copy is for. Bundled via
      the Files section's "dontcopy" entry above, so it has to be
      extracted before it can be copied anywhere. }
    ExtractTemporaryFile('default-config.json');
    CopyFile(ExpandConstant('{tmp}\default-config.json'), ConfigFile, False);
  end;
end;

procedure CurStepChanged(CurStep: TSetupStep);
begin
  if CurStep = ssPostInstall then
  begin
    CreateDefaultConfig();
    if WizardIsTaskSelected('addtopath') then
      EnvAddPath(IsAdminInstallMode, ExpandConstant('{app}'));
  end;
end;

{ ---------- Uninstall: remove the install dir from PATH ---------- }

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
begin
  if CurUninstallStep = usPostUninstall then
    EnvRemovePath(IsAdminInstallMode, ExpandConstant('{app}'));
end;
