# Private Vault With PIN/Password and Authenticator Access

## Summary

Add a separate encrypted Private Vault to the clipboard-history app. First-time users create a PIN/password and configure an authenticator app code. After setup, users can unlock the vault with either the PIN/password or a valid authenticator code. Users can also reset their PIN/password using the authenticator code alone. Normal clipboard items get a `Move to Vault` action, and moved items are removed from normal clipboard history.

## Key Changes

-   Add a Private Vault area with locked, setup, unlock, reset, and unlocked states.
-   First-run setup:
    -   User creates and confirms PIN/password.
    -   App generates a TOTP authenticator secret.
    -   App shows QR code plus manual setup key.
    -   User enters a valid authenticator code to confirm setup.
    -   App creates encrypted vault storage.
-   Unlock behavior:
    -   User can unlock with PIN/password.
    -   User can unlock with authenticator app code.
    -   Vault auto-locks on app restart and after inactivity.
-   Reset behavior:
    -   `Reset PIN/Password` asks for authenticator app code.
    -   If valid, user creates a new PIN/password.
    -   New PIN/password replaces the old one.
-   Clipboard integration:
    -   Add `Move to Vault` action to normal clipboard list items.
    -   If vault is locked, ask user to unlock first.
    -   Save the selected clipboard item into the encrypted vault.
    -   Remove the original item from normal clipboard history after a successful move.
    -   Prevent copied vault secrets from being re-added to normal clipboard history.

## Security and Data Model

-   Store vault entries encrypted at rest.
-   Never store raw PIN/password.
-   Store only a derived verifier/hash for PIN/password validation.
-   Store the TOTP secret encrypted or protected with local app storage mechanisms.
-   Keep plaintext vault data only in memory while unlocked.
-   Hide vault item secret values by default; reveal/copy only on explicit user action.
-   Add failed-attempt handling for PIN/password and authenticator unlock attempts.
-   Treat this as a user-friendly private vault, not a full password-manager-grade recovery model, because authenticator-only unlock/reset requires locally recoverable vault access.

## Test Plan

-   First-time setup requires matching PIN/password confirmation.
-   Authenticator setup fails with invalid code and succeeds with valid code.
-   PIN/password unlock opens the vault.
-   Authenticator code unlock opens the vault.
-   Authenticator code reset allows creating a new PIN/password.
-   Old PIN/password no longer works after reset.
-   `Move to Vault` encrypts and saves the item, then removes it from normal history.
-   Moving fails safely if vault unlock fails.
-   Copying a vault item does not create a new normal clipboard-history entry.
-   Vault is locked after app restart and after inactivity timeout.
