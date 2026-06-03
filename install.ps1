#Requires -Version 5.1
<#
.SYNOPSIS
    安装/卸载菜单脚本。

.DESCRIPTION
    第一步选择“安装”或“卸载”。
    安装时第二步可选择“Codex CLI”、“Claude Code”或“VSCode Claude Code”，选择“Codex CLI”时可选择是否开启 WebSocket，随后输入 API Key。
    选择安装“Codex CLI”时，会自动创建或更新 %USERPROFILE%\.codex\config.toml。
    卸载时第二步可选择“Codex CLI”、“Claude Code”、“VSCode Claude Code”或“全部”。
    使用上下键选择，回车确认，Ctrl+C 终止。
#>

[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

function Show-Menu {
    param(
        [Parameter(Mandatory = $true)]
        [ValidateNotNullOrEmpty()]
        [string]$Title,

        [Parameter(Mandatory = $true)]
        [ValidateNotNullOrEmpty()]
        [string[]]$Options
    )

    $selectedIndex = 0

    while ($true) {
        Clear-Host
        Write-Host "==== $Title ====" -ForegroundColor Cyan
        Write-Host '使用 ↑ / ↓ 选择，Enter 确认，Ctrl+C 终止。'
        Write-Host ''

        for ($i = 0; $i -lt $Options.Count; $i++) {
            if ($i -eq $selectedIndex) {
                Write-Host ("> {0}" -f $Options[$i]) -ForegroundColor Black -BackgroundColor Gray
            }
            else {
                Write-Host ("  {0}" -f $Options[$i])
            }
        }

        $keyInfo = [Console]::ReadKey($true)

        if (($keyInfo.Modifiers -band [ConsoleModifiers]::Control) -and $keyInfo.Key -eq [ConsoleKey]::C) {
            throw (New-Object System.OperationCanceledException '用户按 Ctrl+C 终止。')
        }

        switch ($keyInfo.Key) {
            ([ConsoleKey]::UpArrow) {
                if ($selectedIndex -le 0) {
                    $selectedIndex = $Options.Count - 1
                }
                else {
                    $selectedIndex--
                }
            }
            ([ConsoleKey]::DownArrow) {
                if ($selectedIndex -ge ($Options.Count - 1)) {
                    $selectedIndex = 0
                }
                else {
                    $selectedIndex++
                }
            }
            ([ConsoleKey]::Enter) {
                return $Options[$selectedIndex]
            }
        }
    }
}

function Read-ApiKey {
    param(
        [Parameter(Mandatory = $true)]
        [ValidateSet('Codex CLI', 'Claude Code', 'VSCode Claude Code')]
        [string]$Target
    )

    while ($true) {
        Clear-Host
        Write-Host '==== 输入 API Key ====' -ForegroundColor Cyan
        Write-Host '输入后按 Enter 确认，Ctrl+C 终止。'
        Write-Host ''
        Write-Host 'API Key: ' -NoNewline

        $apiKey = New-Object System.Security.SecureString

        while ($true) {
            $keyInfo = [Console]::ReadKey($true)

            if (($keyInfo.Modifiers -band [ConsoleModifiers]::Control) -and $keyInfo.Key -eq [ConsoleKey]::C) {
                throw (New-Object System.OperationCanceledException '用户按 Ctrl+C 终止。')
            }

            switch ($keyInfo.Key) {
                ([ConsoleKey]::Enter) {
                    Write-Host ''

                    if ($apiKey.Length -gt 0) {
                        $apiKey.MakeReadOnly()
                        return $apiKey
                    }

                    Write-Host 'API Key 不能为空，请重新输入。' -ForegroundColor Yellow
                    Start-Sleep -Seconds 1
                    break
                }
                ([ConsoleKey]::Backspace) {
                    if ($apiKey.Length -gt 0) {
                        $apiKey.RemoveAt($apiKey.Length - 1)
                        Write-Host "`b `b" -NoNewline
                    }
                }
                default {
                    if (-not [char]::IsControl($keyInfo.KeyChar)) {
                        $apiKey.AppendChar($keyInfo.KeyChar)
                        Write-Host '*' -NoNewline
                    }
                }
            }

            if ($keyInfo.Key -eq [ConsoleKey]::Enter -and $apiKey.Length -eq 0) {
                break
            }
        }
    }
}

function Set-CodexCliConfig {
    param(
        [Parameter()]
        [bool]$EnableWebSocket = $false
    )

    $userProfilePath = [Environment]::GetFolderPath('UserProfile')
    if ([string]::IsNullOrWhiteSpace($userProfilePath)) {
        $userProfilePath = $env:USERPROFILE
    }

    if ([string]::IsNullOrWhiteSpace($userProfilePath)) {
        throw '无法确定用户目录。'
    }

    $codexDirectory = Join-Path -Path $userProfilePath -ChildPath '.codex'
    $configPath = Join-Path -Path $codexDirectory -ChildPath 'config.toml'

    if (-not (Test-Path -LiteralPath $codexDirectory -PathType Container)) {
        New-Item -Path $codexDirectory -ItemType Directory -Force | Out-Null
    }

    $webSocketValue = 'false'
    if ($EnableWebSocket) {
        $webSocketValue = 'true'
    }

    $configBlock = @"
# BEGIN install-uninstall-menu Codex CLI config
model_provider = "OpenAI"
model = "gpt-5.5"
review_model = "gpt-5.5"
model_reasoning_effort = "xhigh"
disable_response_storage = true
network_access = "enabled"
windows_wsl_setup_acknowledged = true

[model_providers.OpenAI]
name = "OpenAI"
base_url = "https://api.ruikon.com"
wire_api = "responses"
supports_websockets = $webSocketValue
requires_openai_auth = true

[features]
responses_websockets_v2 = $webSocketValue
goals = true
# END install-uninstall-menu Codex CLI config
"@

    $beginMarker = '# BEGIN install-uninstall-menu Codex CLI config'
    $endMarker = '# END install-uninstall-menu Codex CLI config'
    $newContent = $configBlock.Trim()

    if (Test-Path -LiteralPath $configPath -PathType Leaf) {
        $existingContent = [System.IO.File]::ReadAllText($configPath)
        $escapedBeginMarker = [regex]::Escape($beginMarker)
        $escapedEndMarker = [regex]::Escape($endMarker)
        $managedBlockPattern = "(?s)$escapedBeginMarker.*?$escapedEndMarker"

        if ($existingContent -match $managedBlockPattern) {
            $newContent = [regex]::Replace($existingContent, $managedBlockPattern, $newContent)
        }
        elseif ([string]::IsNullOrWhiteSpace($existingContent)) {
            $newContent = $newContent
        }
        else {
            $newContent = $existingContent.TrimEnd() + [Environment]::NewLine + [Environment]::NewLine + $newContent
        }
    }

    $utf8NoBom = New-Object System.Text.UTF8Encoding($false)
    [System.IO.File]::WriteAllText($configPath, $newContent + [Environment]::NewLine, $utf8NoBom)

    return $configPath
}

function Set-CodexCliAuthJson {
    param(
        [Parameter(Mandatory = $true)]
        [ValidateNotNull()]
        [securestring]$ApiKey
    )

    $userProfilePath = [Environment]::GetFolderPath('UserProfile')
    if ([string]::IsNullOrWhiteSpace($userProfilePath)) {
        $userProfilePath = $env:USERPROFILE
    }

    if ([string]::IsNullOrWhiteSpace($userProfilePath)) {
        throw '无法确定用户目录。'
    }

    $codexDirectory = Join-Path -Path $userProfilePath -ChildPath '.codex'
    $authPath = Join-Path -Path $codexDirectory -ChildPath 'auth.json'

    if (-not (Test-Path -LiteralPath $codexDirectory -PathType Container)) {
        New-Item -Path $codexDirectory -ItemType Directory -Force | Out-Null
    }

    $bstr = [System.IntPtr]::Zero
    try {
        $bstr = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($ApiKey)
        $plainTextApiKey = [Runtime.InteropServices.Marshal]::PtrToStringBSTR($bstr)

        $authObject = @{ OPENAI_API_KEY = $plainTextApiKey }
        $json = $authObject | ConvertTo-Json -Depth 2
        $utf8NoBom = New-Object System.Text.UTF8Encoding($false)
        [System.IO.File]::WriteAllText($authPath, $json + [Environment]::NewLine, $utf8NoBom)
    }
    finally {
        if ($bstr -ne [System.IntPtr]::Zero) {
            [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($bstr)
        }
    }

    return $authPath
}

function Set-ClaudeCodeUserEnvironmentVariables {
    param(
        [Parameter(Mandatory = $true)]
        [ValidateNotNull()]
        [securestring]$ApiKey
    )

    $bstr = [System.IntPtr]::Zero
    try {
        $bstr = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($ApiKey)
        $plainTextApiKey = [Runtime.InteropServices.Marshal]::PtrToStringBSTR($bstr)

        $environmentEntries = @(
            @{ Name = 'ANTHROPIC_BASE_URL'; Value = 'https://api.ruikon.com' },
            @{ Name = 'ANTHROPIC_AUTH_TOKEN'; Value = $plainTextApiKey },
            @{ Name = 'CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC'; Value = '1' }
        )

        foreach ($entry in $environmentEntries) {
            Write-Host ("正在写入用户环境变量：{0} ..." -f $entry.Name) -ForegroundColor Yellow
            [Environment]::SetEnvironmentVariable($entry.Name, $entry.Value, [EnvironmentVariableTarget]::User)
            [Environment]::SetEnvironmentVariable($entry.Name, $entry.Value, [EnvironmentVariableTarget]::Process)
        }
    }
    finally {
        if ($bstr -ne [System.IntPtr]::Zero) {
            [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($bstr)
        }
    }
}

function Set-VSCodeClaudeCodeSettings {
    param(
        [Parameter(Mandatory = $true)]
        [ValidateNotNull()]
        [securestring]$ApiKey
    )

    $userProfilePath = [Environment]::GetFolderPath('UserProfile')
    if ([string]::IsNullOrWhiteSpace($userProfilePath)) {
        $userProfilePath = $env:USERPROFILE
    }

    if ([string]::IsNullOrWhiteSpace($userProfilePath)) {
        throw '无法确定用户目录。'
    }

    $claudeDirectory = Join-Path -Path $userProfilePath -ChildPath '.claude'
    $settingsPath = Join-Path -Path $claudeDirectory -ChildPath 'settings.json'

    if (-not (Test-Path -LiteralPath $claudeDirectory -PathType Container)) {
        New-Item -Path $claudeDirectory -ItemType Directory -Force | Out-Null
    }

    $bstr = [System.IntPtr]::Zero
    try {
        $bstr = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($ApiKey)
        $plainTextApiKey = [Runtime.InteropServices.Marshal]::PtrToStringBSTR($bstr)

        $settingsObject = [ordered]@{
            env = [ordered]@{
                ANTHROPIC_BASE_URL = 'https://api.ruikon.com'
                ANTHROPIC_AUTH_TOKEN = $plainTextApiKey
                CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC = '1'
                CLAUDE_CODE_ATTRIBUTION_HEADER = '0'
            }
        }

        $json = $settingsObject | ConvertTo-Json -Depth 4
        $utf8NoBom = New-Object System.Text.UTF8Encoding($false)
        [System.IO.File]::WriteAllText($settingsPath, $json + [Environment]::NewLine, $utf8NoBom)
    }
    finally {
        if ($bstr -ne [System.IntPtr]::Zero) {
            [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($bstr)
        }
    }

    return $settingsPath
}

function Invoke-Install {
    param(
        [Parameter(Mandatory = $true)]
        [ValidateSet('Codex CLI', 'Claude Code', 'VSCode Claude Code')]
        [string]$Target,

        [Parameter(Mandatory = $true)]
        [ValidateNotNull()]
        [securestring]$ApiKey,

        [Parameter()]
        [bool]$EnableWebSocket = $false
    )

    Clear-Host
    Write-Host "已选择：安装 $Target" -ForegroundColor Green
    Write-Host '已输入 API Key。' -ForegroundColor Green

    if ($Target -eq 'Codex CLI') {
        if ($EnableWebSocket) {
            $webSocketStatus = '开启'
        }
        else {
            $webSocketStatus = '不开启'
        }

        $codexConfigPath = Set-CodexCliConfig -EnableWebSocket $EnableWebSocket
        $codexAuthPath = Set-CodexCliAuthJson -ApiKey $ApiKey
        Write-Host "WebSocket：$webSocketStatus" -ForegroundColor Green
        Write-Host "已更新配置：$codexConfigPath" -ForegroundColor Green
        Write-Host "已更新认证：$codexAuthPath" -ForegroundColor Green
    }
    elseif ($Target -eq 'Claude Code') {
        Write-Host '正在写入 Claude Code 用户环境变量，请稍候...' -ForegroundColor Yellow
        Set-ClaudeCodeUserEnvironmentVariables -ApiKey $ApiKey
        Write-Host '已写入 Claude Code 用户环境变量。' -ForegroundColor Green
    }
    elseif ($Target -eq 'VSCode Claude Code') {
        Write-Host '正在写入 VSCode Claude Code 配置，请稍候...' -ForegroundColor Yellow
        $vscodeClaudeSettingsPath = Set-VSCodeClaudeCodeSettings -ApiKey $ApiKey
        Write-Host "已更新配置：$vscodeClaudeSettingsPath" -ForegroundColor Green
    }

    Write-Verbose ("API Key 长度：{0}" -f $ApiKey.Length)
    # TODO: 在这里添加 $Target 的安装逻辑，按需使用 $ApiKey 和 $EnableWebSocket。
}

function Get-CurrentUserProfilePath {
    $userProfilePath = [Environment]::GetFolderPath('UserProfile')
    if ([string]::IsNullOrWhiteSpace($userProfilePath)) {
        $userProfilePath = $env:USERPROFILE
    }

    if ([string]::IsNullOrWhiteSpace($userProfilePath)) {
        throw '无法确定用户目录。'
    }

    return $userProfilePath
}

function Remove-FileIfExists {
    param(
        [Parameter(Mandatory = $true)]
        [ValidateNotNullOrEmpty()]
        [string]$Path
    )

    if (Test-Path -LiteralPath $Path -PathType Leaf) {
        Remove-Item -LiteralPath $Path -Force
        Write-Host "已删除：$Path" -ForegroundColor Green
    }
    else {
        Write-Host "文件不存在，跳过：$Path" -ForegroundColor DarkYellow
    }
}

function Uninstall-CodexCli {
    $userProfilePath = Get-CurrentUserProfilePath
    $codexDirectory = Join-Path -Path $userProfilePath -ChildPath '.codex'

    Write-Host '正在卸载 Codex CLI 配置...' -ForegroundColor Yellow
    Remove-FileIfExists -Path (Join-Path -Path $codexDirectory -ChildPath 'config.toml')
    Remove-FileIfExists -Path (Join-Path -Path $codexDirectory -ChildPath 'auth.json')
}

function Uninstall-ClaudeCode {
    $environmentVariableNames = @(
        'ANTHROPIC_BASE_URL',
        'ANTHROPIC_AUTH_TOKEN',
        'CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC'
    )

    Write-Host '正在删除 Claude Code 用户环境变量...' -ForegroundColor Yellow

    foreach ($name in $environmentVariableNames) {
        $userValue = [Environment]::GetEnvironmentVariable($name, [EnvironmentVariableTarget]::User)
        $processValue = [Environment]::GetEnvironmentVariable($name, [EnvironmentVariableTarget]::Process)
        $removed = $false

        if ($null -ne $userValue) {
            [Environment]::SetEnvironmentVariable($name, $null, [EnvironmentVariableTarget]::User)
            Write-Host "已删除用户环境变量：$name" -ForegroundColor Green
            $removed = $true
        }

        if ($null -ne $processValue) {
            [Environment]::SetEnvironmentVariable($name, $null, [EnvironmentVariableTarget]::Process)
            if (-not $removed) {
                Write-Host "已删除当前进程环境变量：$name" -ForegroundColor Green
            }
        }

        if ($null -eq $userValue -and $null -eq $processValue) {
            Write-Host "环境变量不存在，跳过：$name" -ForegroundColor DarkYellow
        }
    }
}

function Uninstall-VSCodeClaudeCode {
    $userProfilePath = Get-CurrentUserProfilePath
    $claudeDirectory = Join-Path -Path $userProfilePath -ChildPath '.claude'
    $settingsPath = Join-Path -Path $claudeDirectory -ChildPath 'settings.json'

    Write-Host '正在卸载 VSCode Claude Code 配置...' -ForegroundColor Yellow
    Remove-FileIfExists -Path $settingsPath
}

function Invoke-Uninstall {
    param(
        [Parameter(Mandatory = $true)]
        [ValidateSet('Codex CLI', 'Claude Code', 'VSCode Claude Code', '全部')]
        [string]$Target
    )

    Clear-Host
    Write-Host "已选择：卸载 $Target" -ForegroundColor Green

    switch ($Target) {
        'Codex CLI' {
            Uninstall-CodexCli
        }
        'Claude Code' {
            Uninstall-ClaudeCode
        }
        'VSCode Claude Code' {
            Uninstall-VSCodeClaudeCode
        }
        '全部' {
            Uninstall-CodexCli
            Uninstall-ClaudeCode
            Uninstall-VSCodeClaudeCode
        }
    }

    Write-Host '卸载处理完成。' -ForegroundColor Green
}

try {
    $previousTreatControlCAsInput = [Console]::TreatControlCAsInput
    [Console]::TreatControlCAsInput = $true

    $action = Show-Menu -Title '请选择操作' -Options @('安装', '卸载')

    switch ($action) {
        '安装' {
            $target = Show-Menu -Title '请选择安装项目' -Options @('Codex CLI', 'Claude Code', 'VSCode Claude Code')
            $enableWebSocket = $false

            if ($target -eq 'Codex CLI') {
                $webSocketChoice = Show-Menu -Title '是否开启 WebSocket' -Options @('不开启', '开启')
                $enableWebSocket = $webSocketChoice -eq '开启'
            }

            $apiKey = Read-ApiKey -Target $target
            Invoke-Install -Target $target -ApiKey $apiKey -EnableWebSocket $enableWebSocket
        }
        '卸载' {
            $target = Show-Menu -Title '请选择卸载项目' -Options @('Codex CLI', 'Claude Code', 'VSCode Claude Code', '全部')
            Invoke-Uninstall -Target $target
        }
    }
}
catch [System.OperationCanceledException] {
    Clear-Host
    Write-Host '已终止。' -ForegroundColor Yellow
    exit 130
}
catch {
    Write-Error "脚本执行失败：$($_.Exception.Message)"
    exit 1
}
finally {
    if (Get-Variable -Name previousTreatControlCAsInput -Scope Local -ErrorAction SilentlyContinue) {
        [Console]::TreatControlCAsInput = $previousTreatControlCAsInput
    }
}
