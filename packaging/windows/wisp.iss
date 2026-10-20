#ifndef AppVersion
  #error AppVersion must be supplied by build-installer.ps1
#endif
#ifndef PayloadDir
  #error PayloadDir must be supplied by build-installer.ps1
#endif
#ifndef ReleaseId
  #error ReleaseId must be supplied by build-installer.ps1
#endif

[Setup]
AppId={{9ACD2363-C14C-4D5B-A38A-9FA4D819E5CF}
AppName=wisp
AppVersion={#AppVersion}
AppPublisher=cmuty
AppPublisherURL=https://github.com/cmuty/wisp
AppSupportURL=https://github.com/cmuty/wisp/issues
AppUpdatesURL=https://github.com/cmuty/wisp/releases
DefaultDirName={localappdata}\Programs\wisp
PrivilegesRequired=lowest
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
MinVersion=10.0.17763
WizardStyle=modern
DisableProgramGroupPage=yes
DisableWelcomePage=no
CloseApplications=no
RestartApplications=no
ChangesEnvironment=yes
Compression=lzma2
SolidCompression=yes
OutputDir={#PayloadDir}
OutputBaseFilename=wisp-{#AppVersion}-setup-x64
UninstallDisplayName=wisp
VersionInfoVersion={#AppVersion}
LicenseFile=..\..\LICENSE

[Languages]
Name: english; MessagesFile: compiler:Default.isl
Name: russian; MessagesFile: compiler:Languages\Russian.isl

[Messages]
english.WelcomeLabel2=A little room for sound.%n%nSetup installs wisp for your Windows account and makes the wisp command available in new terminals.%n%nInternet access is required to fetch verified audio components from their upstream projects. Your listening profile and collections are kept when updating or uninstalling.
russian.WelcomeLabel2=Немного места для звука.%n%nУстановщик добавит wisp для твоей учётной записи Windows и команду wisp в новых терминалах.%n%nДля загрузки проверенных аудиокомпонентов нужен интернет. Профиль и коллекции сохраняются при обновлении и удалении.

[CustomMessages]
english.ProvisionFailed=Audio setup or file installation failed. Check your connection, close wisp, and try again. Installer log: %1
russian.ProvisionFailed=Не удалось установить файлы или аудиокомпоненты. Проверь интернет, закрой wisp и повтори установку. Журнал: %1

[Files]
Source: "{#PayloadDir}\wisp-{#AppVersion}-windows-amd64.zip"; DestName: wisp.zip; Flags: dontcopy
Source: "{#PayloadDir}\install.ps1"; Flags: dontcopy
Source: "provision.ps1"; Flags: dontcopy

[Icons]
Name: "{userprograms}\wisp"; Filename: "{app}\bin\wisp.cmd"; WorkingDir: "{userprofile}"

[UninstallDelete]
Type: files; Name: "{app}\bin\wisp.cmd"
Type: dirifempty; Name: "{app}\bin"
Type: filesandordirs; Name: "{app}\releases\{#ReleaseId}"
Type: dirifempty; Name: "{app}\releases"
Type: dirifempty; Name: "{app}"

[Code]
procedure UpdateUserPath(AddEntry: Boolean);
var
  OldPath, NewPath, Entry, BinDir: String;
  Separator: Integer;
begin
  RegQueryStringValue(HKCU, 'Environment', 'Path', OldPath);
  BinDir := ExpandConstant('{app}\bin');
  NewPath := '';
  while OldPath <> '' do begin
    Separator := Pos(';', OldPath);
    if Separator = 0 then Separator := Length(OldPath) + 1;
    Entry := Copy(OldPath, 1, Separator - 1);
    Delete(OldPath, 1, Separator);
    if (Entry <> '') and (CompareText(Entry, BinDir) <> 0) then begin
      if NewPath <> '' then NewPath := NewPath + ';';
      NewPath := NewPath + Entry;
    end;
  end;
  if AddEntry then begin
    if NewPath <> '' then NewPath := NewPath + ';';
    NewPath := NewPath + BinDir;
  end;
  if not RegWriteExpandStringValue(HKCU, 'Environment', 'Path', NewPath) then
    RaiseException('Could not update the user PATH.');
end;

function PrepareToInstall(var NeedsRestart: Boolean): String;
var
  ExitCode: Integer;
  Arguments, LogPath: String;
begin
  ExtractTemporaryFile('wisp.zip');
  ExtractTemporaryFile('install.ps1');
  ExtractTemporaryFile('provision.ps1');
  LogPath := ExpandConstant('{localappdata}\wisp\installer.log');
  Arguments := '-NoLogo -NoProfile -NonInteractive -ExecutionPolicy Bypass -File "' +
    ExpandConstant('{tmp}\provision.ps1') + '" -ArchivePath "' +
    ExpandConstant('{tmp}\wisp.zip') + '" -InstallDir "' +
    ExpandConstant('{app}') + '" -LogPath "' + LogPath + '"';
  if not Exec(ExpandConstant('{sys}\WindowsPowerShell\v1.0\powershell.exe'),
    Arguments, '', SW_HIDE, ewWaitUntilTerminated, ExitCode) or (ExitCode <> 0) then
    Result := FmtMessage(CustomMessage('ProvisionFailed'), [LogPath])
  else Result := '';
end;

procedure CurStepChanged(CurStep: TSetupStep);
begin
  if CurStep = ssPostInstall then UpdateUserPath(True);
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
begin
  if CurUninstallStep = usPostUninstall then UpdateUserPath(False);
end;
