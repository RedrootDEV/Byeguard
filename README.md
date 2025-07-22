# Byeguard

**Byeguard** is a lightweight Windows utility to enable or disable Riot Vanguard, the anti-cheat system for Valorant and other Riot games.

---

## Why Byeguard?

Riot Vanguard runs persistently on your system, monitoring your PC continuously—even when you're not playing Valorant or League of Legends. Given concerns about privacy and resource usage, **Byeguard** allows you to easily disable Vanguard when not needed, and re-enable it when you want to play.

This tool provides a direct way to:

- Disable Vanguard services and processes.
- Rename Vanguard files to prevent it from starting.
- Re-enable Vanguard safely.
- Prompt for a system reboot if needed.

---

## How to Use

1. Download the latest release executable (`Byeguard.exe`) from the [Releases](https://github.com/RedrootDEV/Byeguard/releases) section.

2. Run `Byeguard.exe` **as administrator** (it will prompt to elevate if needed).

3. The program will detect if Vanguard is currently enabled or disabled.

4. Confirm your choice when prompted to disable or enable Vanguard.

5. If enabling Vanguard, you will be asked whether you want to reboot your system immediately for changes to take effect.

---

## Important Notes

- This tool **requires administrative privileges** to manage system services and files.

- Modifying or disabling Vanguard may impact your ability to play Riot games.

- Use at your own risk. Always ensure you have backups of important data.

- This project is provided as-is, without warranty.

---

## Building from Source

If you prefer to build from source:

```bash
go build -o Byeguard.exe main.go
```

Make sure you have Go installed and set up properly.

## License

MIT License — see the [LICENSE](LICENSE) file for details.

## Disclaimer

This tool is for personal use to control the Vanguard anti-cheat service on your own machine. It is not intended for cheating, hacking, or bypassing anti-cheat protections in an unauthorized manner.

## Contact

For issues or suggestions, please open an issue on this repository.

---

Byeguard — Take control of Riot Vanguard privacy and performance.