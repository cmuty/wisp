[CmdletBinding()]
param(
    [int]$WispProcessId = 0,
    [ValidateRange(5,300)][int]$DurationSeconds = 30,
    [string]$Context = 'Existing wisp session',
    [string]$OutputPath = '.preview-cache/memory-sample.json'
)
$ErrorActionPreference = 'Stop'
if (!$WispProcessId) {
    $instances = @(Get-Process wisp -ErrorAction SilentlyContinue)
    if ($instances.Count -ne 1) { throw 'Start one wisp instance or specify -WispProcessId.' }
    $WispProcessId = $instances[0].Id
}
$rootProcess = Get-Process -Id $WispProcessId
if ($rootProcess.ProcessName -ne 'wisp') { throw 'The selected process must be wisp.' }
$started = $rootProcess.StartTime
$version = (& $rootProcess.Path --version | Out-String).Trim()
$begin = [DateTime]::UtcNow
$samples = [Collections.Generic.List[object]]::new()
$clock = [Diagnostics.Stopwatch]::StartNew()
while ($clock.Elapsed.TotalSeconds -lt $DurationSeconds) {
    $rootNow = Get-Process -Id $WispProcessId
    if ($rootNow.StartTime -ne $started) { throw 'The selected wisp process restarted during measurement.' }
    $tree = @(Get-CimInstance Win32_Process | Select-Object ProcessId,ParentProcessId,Name)
    $descendants = [Collections.Generic.HashSet[int]]::new()
    [void]$descendants.Add($WispProcessId)
    do {
        $changed = $false
        foreach ($entry in $tree) {
            if ($descendants.Contains([int]$entry.ParentProcessId) -and $descendants.Add([int]$entry.ProcessId)) { $changed = $true }
        }
    } while ($changed)
    $helpers = @()
    foreach ($entry in $tree) {
        if ($descendants.Contains([int]$entry.ProcessId) -and $entry.Name -in @('mpv.exe','yt-dlp.exe','deno.exe','node.exe')) {
            $helper = Get-Process -Id $entry.ProcessId -ErrorAction SilentlyContinue
            if ($helper) {
                $helpers += [PSCustomObject]@{name=$entry.Name;workingSetBytes=$helper.WorkingSet64;privateBytes=$helper.PrivateMemorySize64}
            }
        }
    }
    $helperWorkingSet = [long]($helpers | Measure-Object workingSetBytes -Sum).Sum
    $samples.Add([PSCustomObject]@{
        seconds=[Math]::Round($clock.Elapsed.TotalSeconds,3)
        wispWorkingSetBytes=$rootNow.WorkingSet64
        wispPrivateBytes=$rootNow.PrivateMemorySize64
        helperWorkingSetBytes=$helperWorkingSet
        combinedWorkingSetBytes=$rootNow.WorkingSet64+$helperWorkingSet
        helpers=$helpers
    })
    Start-Sleep -Milliseconds 1000
}
$clock.Stop()
$summary = [ordered]@{}
foreach ($metric in @('wispWorkingSetBytes','helperWorkingSetBytes','combinedWorkingSetBytes','wispPrivateBytes')) {
    $statistics = $samples | Measure-Object -Property $metric -Average -Minimum -Maximum
    $summary[$metric] = [ordered]@{mean=[long]$statistics.Average;min=[long]$statistics.Minimum;max=[long]$statistics.Maximum}
}
$result = [ordered]@{
    version=$version
    measuredAtUtc=$begin.ToString('o')
    windowsVersion=[Environment]::OSVersion.Version.ToString()
    architecture=if ([Environment]::Is64BitOperatingSystem) {'x64'} else {'x86'}
    context=$Context
    durationSeconds=[Math]::Round($clock.Elapsed.TotalSeconds,2)
    sampleCount=$samples.Count
    metric='Process working set in bytes; combined values sum process working sets, including shared pages.'
    exclusions='Terminal host and measurement tool; helpers absent between samples may be missed.'
    summary=$summary
    samples=$samples.ToArray()
}
$destination = [IO.Path]::GetFullPath($OutputPath)
[void][IO.Directory]::CreateDirectory([IO.Path]::GetDirectoryName($destination))
[IO.File]::WriteAllText($destination,($result | ConvertTo-Json -Depth 8),[Text.UTF8Encoding]::new($false))
Write-Output "Saved $($samples.Count) samples over $($result.durationSeconds) seconds to $destination"
foreach ($metric in $summary.Keys) {
    Write-Output ('{0}: mean {1:N1} MiB; peak {2:N1} MiB' -f $metric,($summary[$metric].mean/1MB),($summary[$metric].max/1MB))
}
