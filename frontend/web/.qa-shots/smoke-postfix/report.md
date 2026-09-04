# Prova smoke-postfix QA

**Target:** http://localhost:3000 (Turkish UI)
**Viewport checks:** mobile 390×844, desktop 1440
**Result:** 8 passed / 0 failed (of 8)

| # | Status | Detail |
|---|--------|--------|
| 1 | **PASS** | Statuses: {"/":200,"/login":200,"/register":200,"/verify":200,"/sessions":200} |
| 2 | **PASS** | OK: trigger+header, cards=4 values=["12","%64","3","2","Mevzuat değişti, 2 sertifika eski rubrikle verildi"], no persistent sidebar, readable |
| 3 | **PASS** | OK: table 1086>340, tabs wrap found, no full sidebar |
| 4 | **PASS** | OK: stayed on /login, error has Geçerli bir e-posta, submit BUTTON |
| 5 | **PASS** | OK: errors=["*","Ad soyad gerekli","Geçerli bir e-posta girin","Doğrulama kodu gönder"], no Link CTA to /verify |
| 6 | **PASS** | OK: error "6 haneli kodu girin", Doğrula BUTTON type=submit |
| 7 | **PASS** | none |
| 8 | **PASS** | OK: sidebar visible width=220 |

## Screenshots
- `home-390.png`, `sessions-390.png`, `login-error.png`, `register-error.png`, `verify-error.png`, `home-1440.png`
- Saved under `/Users/eminkutlu/dev/Prova/frontend/web/.qa-shots/smoke-postfix` and `/tmp/prova-qa/smoke-postfix`

## Notes
- Short post-fix smoke only; no auth login flow beyond empty-form validation.
- Favicon 4xx/5xx excluded from failed-request check.
