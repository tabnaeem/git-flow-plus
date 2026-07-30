# Upgrade Guide

How to move an existing Git Flow Plus install to a newer version, and
how to remove it entirely — by platform and install method. In every
case, your data is preserved: no upgrade or uninstall path deletes a
repository's `.gitflowplus/` directory (that's just files in your own
repo) or the seeded config/logs directories described in
[WindowsInstallation.md](WindowsInstallation.md#seeded-files)/
[Packaging.md](Packaging.md#macos-pkg).

## Windows — `.exe` installer

Just run the new version's installer. It detects the existing install
(via its own installer ID, regardless of version) and **silently
uninstalls it first**, before laying down the new files — verified
behavior, not a theoretical claim; see
[Packaging.md](Packaging.md#the-exe-installer). No separate uninstall
step needed, and your `PATH` entry, seeded config, and logs directory
all survive untouched.

To uninstall entirely instead of upgrading, see
[WindowsInstallation.md#silent-uninstall](WindowsInstallation.md#silent-uninstall)
or use **Settings → Apps → Git Flow Plus → Uninstall** for the
interactive route.

## Windows — `.msi`

MSI's native `MajorUpgrade` mechanism handles this — installing a newer
`.msi` automatically removes the older one first, same as the `.exe`.
No manual uninstall step:

```powershell
msiexec /i git-flow-plus-<new-version>-windows-x64.msi /quiet /norestart
```

To remove entirely: `msiexec /x git-flow-plus-<version>-windows-x64.msi /quiet /norestart`,
or **Settings → Apps**.

## Linux — `.deb`/`.rpm`

Reinstalling over an existing package is the native upgrade path for
both formats:

```bash
sudo dpkg -i git-flow-plus-<new-version>-amd64.deb   # Debian/Ubuntu
sudo rpm -U git-flow-plus-<new-version>-x86_64.rpm    # Fedora/RHEL
```

To remove: `sudo dpkg -r git-flow-plus` / `sudo rpm -e git-flow-plus`.

## macOS — `.pkg`

Run the new version's `.pkg` the same way you installed the first one —
`pkgbuild`-based installers overwrite in place:

```bash
sudo installer -pkg git-flow-plus-<new-version>-macos-universal.pkg -target /
```

macOS `.pkg` installers don't register an uninstaller; to remove:

```bash
sudo rm /usr/local/bin/git-flow-plus /usr/local/bin/git-flow
```

(Your seeded config at `~/Library/Application Support/GitFlowPlus` is
untouched — remove it manually too if you want a completely clean
slate.)

## Raw archive (any platform)

There's no installer to run — just overwrite the binary in place with
the new archive's copy:

```bash
tar -xzf git-flow-plus-<new-version>-<platform>.tar.gz -C /usr/local/bin --strip-components=0 git-flow-plus
```

(Windows equivalent: extract the new `.zip` over the old location.) The
`git-flow` symlink/copy, if you created one per
[Installation.md](Installation.md), doesn't need recreating — it already
points at (or is) the binary you just replaced.

## Verifying an upgrade worked

```bash
git flow version
```

Confirms the `Version` line matches what you just installed. Follow
with `git flow doctor` to confirm nothing else regressed.

## Downgrading

Not a supported path on any platform — Windows' MSI explicitly refuses
a downgrade (`DowngradeErrorMessage` in
[installer/windows/gitflowplus.wxs](installer/windows/gitflowplus.wxs)),
and the Inno `.exe`/`.deb`/`.rpm`/`.pkg` installers don't check version
order at all, so installing an older version over a newer one works
mechanically but isn't tested or recommended. If you need an older
version, uninstall first, then install the old one from its own release
page.

## See also

- [Installation.md](Installation.md) — first-time install
- [ReleaseProcess.md](ReleaseProcess.md) — how new versions get published
- [Troubleshooting.md](Troubleshooting.md) — if an upgrade doesn't behave as described here
