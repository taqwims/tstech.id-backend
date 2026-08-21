package service

import (
	"fmt"
	"log"
	"net/smtp"
	"strings"

	"github.com/tstech/backend/internal/config"
)

type EmailService struct {
	cfg *config.Config
}

func NewEmailService(cfg *config.Config) *EmailService {
	return &EmailService{cfg: cfg}
}

// SendOrderConfirmation sends order confirmation email with optional login credentials
func (s *EmailService) SendOrderConfirmation(to, customerName, orderNumber string, amount int64, defaultPassword string, isNewAccount bool) error {
	subject := fmt.Sprintf("Konfirmasi Pesanan #%s - TsTech", orderNumber)

	accountInfoHTML := ""
	if isNewAccount && defaultPassword != "" {
		accountInfoHTML = fmt.Sprintf(`
			<div style="background-color: #f1f5f9; border-left: 4px solid #6366f1; padding: 16px; margin: 20px 0; border-radius: 8px; font-family: sans-serif;">
				<h3 style="color: #0f172a; margin-top: 0;">📊 Akses Dashboard Klien Anda</h3>
				<p style="color: #334155; margin-bottom: 12px;">
					Akun dashboard Anda telah dibuat secara otomatis agar Anda dapat memantau progres pengerjaan software, timeline milestone, dan riwayat revisi secara realtime.
				</p>
				<table style="color: #334155; font-size: 14px;">
					<tr>
						<td style="padding: 4px 8px 4px 0;"><strong>URL Login:</strong></td>
						<td><a href="%s/login" style="color: #6366f1; font-weight: bold;">%s/login</a></td>
					</tr>
					<tr>
						<td style="padding: 4px 8px 4px 0;"><strong>Email:</strong></td>
						<td><code>%s</code></td>
					</tr>
					<tr>
						<td style="padding: 4px 8px 4px 0;"><strong>Password Sementara:</strong></td>
						<td><code style="background: #e2e8f0; padding: 2px 6px; border-radius: 4px; font-weight: bold; color: #1e293b;">%s</code></td>
					</tr>
				</table>
				<p style="color: #64748b; font-size: 12px; margin-top: 12px; margin-bottom: 0;">
					*Anda dapat mengganti kata sandi ini kapan saja setelah login pada menu Profil Akun.
				</p>
			</div>
		`, s.cfg.AppURL, s.cfg.AppURL, to, defaultPassword)
	}

	body := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; color: #334155; line-height: 1.6;">
			<h2 style="color: #0f172a;">Halo, %s!</h2>
			<p>Terima kasih telah mempercayakan proyek software Anda kepada <strong>TsTech</strong>. Pesanan Anda telah berhasil tercatat di sistem kami.</p>
			
			<div style="background: #ffffff; border: 1px solid #e2e8f0; border-radius: 8px; padding: 16px; margin: 16px 0;">
				<h4 style="margin-top: 0; color: #0f172a;">Detail Pesanan:</h4>
				<ul style="margin: 0; padding-left: 20px;">
					<li><strong>Nomor Pesanan:</strong> #%s</li>
					<li><strong>Total Biaya:</strong> Rp %d</li>
				</ul>
			</div>

			%s

			<p>Tim engineering kami akan segera menghubungi Anda melalui WhatsApp untuk memulai tahap briefing requirement.</p>
			<br>
			<p style="color: #64748b; font-size: 14px;">Salam hangat,<br><strong>Tim TsTech</strong><br><a href="%s" style="color: #6366f1;">%s</a></p>
		</div>
	`, customerName, orderNumber, amount, accountInfoHTML, s.cfg.AppURL, s.cfg.AppURL)

	return s.sendEmail(to, subject, body)
}

// SendConsultationConfirmation sends confirmation for consultation request
func (s *EmailService) SendConsultationConfirmation(to, name string) error {
	subject := "Permintaan Konsultasi Diterima - TsTech"
	body := fmt.Sprintf(`
		<h2>Halo, %s!</h2>
		<p>Permintaan konsultasi Anda telah kami terima.</p>
		<p>Tim kami akan segera menghubungi Anda untuk menjadwalkan sesi konsultasi.</p>
		<br>
		<p>Salam,<br>Tim TsTech</p>
	`, name)

	return s.sendEmail(to, subject, body)
}

// SendPaymentReceipt sends payment receipt confirmation to customer
func (s *EmailService) SendPaymentReceipt(to, customerName, invoiceNumber, paymentMethod string, amount int64, paymentType, projectTitle string) error {
	subject := fmt.Sprintf("Bukti Pembayaran Berhasil #%s - TsTech", invoiceNumber)
	typeDesc := "Pembayaran DP (Uang Muka)"
	if paymentType == "pelunasan" {
		typeDesc = "Pelunasan Tagihan Proyek"
	} else if paymentType == "full_payment" {
		typeDesc = "Pembayaran Penuh"
	}

	body := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; color: #334155; line-height: 1.6;">
			<div style="background: linear-gradient(135deg, #4f46e5 0%%, #06b6d4 100%%); padding: 24px; border-radius: 12px 12px 0 0; text-align: center; color: white;">
				<h1 style="margin: 0; font-size: 24px;">TsTech</h1>
				<p style="margin: 6px 0 0; opacity: 0.9; font-size: 14px;">Pembayaran Anda Telah Berhasil Diterima ✓</p>
			</div>
			<div style="background: #ffffff; border: 1px solid #e2e8f0; border-top: none; border-radius: 0 0 12px 12px; padding: 24px;">
				<h2 style="color: #0f172a; margin-top: 0;">Halo, %s!</h2>
				<p>Pembayaran Anda sebesar <strong>Rp %d</strong> untuk invoice <strong>#%s</strong> telah berhasil kami terima dan diverifikasi secara otomatis.</p>
				
				<div style="background: #f8fafc; border: 1px solid #e2e8f0; border-radius: 8px; padding: 16px; margin: 20px 0;">
					<table style="width: 100%%; font-size: 14px; color: #334155;">
						<tr>
							<td style="padding: 6px 0;"><strong>Nomor Invoice:</strong></td>
							<td style="text-align: right; font-family: monospace; font-weight: bold; color: #6366f1;">%s</td>
						</tr>
						<tr>
							<td style="padding: 6px 0;"><strong>Tipe Pembayaran:</strong></td>
							<td style="text-align: right;">%s</td>
						</tr>
						<tr>
							<td style="padding: 6px 0;"><strong>Metode Gateway:</strong></td>
							<td style="text-align: right; text-transform: uppercase;">%s</td>
						</tr>
						<tr>
							<td style="padding: 6px 0;"><strong>Proyek:</strong></td>
							<td style="text-align: right; font-weight: bold;">%s</td>
						</tr>
						<tr style="border-top: 1px solid #cbd5e1;">
							<td style="padding: 8px 0; font-size: 16px;"><strong>Total Dibayar:</strong></td>
							<td style="text-align: right; font-size: 16px; font-weight: bold; color: #10b981;">Rp %d</td>
						</tr>
					</table>
				</div>

				<p>Proyek Anda kini telah aktif. Anda dapat memantau perkembangan software dan berdiskusi dengan tim kami di dashboard:</p>
				<div style="text-align: center; margin: 24px 0;">
					<a href="%s/dashboard" style="background: #6366f1; color: white; padding: 12px 28px; text-decoration: none; border-radius: 8px; font-weight: bold; display: inline-block;">Buka Dashboard Klien &rarr;</a>
				</div>
				
				<p style="color: #64748b; font-size: 13px; margin-bottom: 0;">Jika ada pertanyaan, silakan hubungi tim kami via WhatsApp di nomor resmi TsTech.</p>
			</div>
		</div>
	`, customerName, amount, invoiceNumber, invoiceNumber, typeDesc, paymentMethod, projectTitle, amount, s.cfg.AppURL)

	return s.sendEmail(to, subject, body)
}

// SendAdminNotification notifies admin about new submissions
func (s *EmailService) SendAdminNotification(subject, body string) error {
	return s.sendEmail(s.cfg.SMTPUser, subject, body)
}

func (s *EmailService) sendEmail(to, subject, body string) error {
	if s.cfg.SMTPUser == "" || s.cfg.SMTPPassword == "" {
		log.Printf("⚠️ SMTP not configured, skipping email to %s: %s", to, subject)
		return nil
	}

	from := s.cfg.SMTPFromEmail
	mime := "MIME-version: 1.0;\r\nContent-Type: text/html; charset=\"UTF-8\";\r\n"

	msg := strings.Join([]string{
		"From: " + s.cfg.SMTPFromName + " <" + from + ">",
		"To: " + to,
		"Subject: " + subject,
		mime,
		"",
		body,
	}, "\r\n")

	addr := fmt.Sprintf("%s:%d", s.cfg.SMTPHost, s.cfg.SMTPPort)
	auth := smtp.PlainAuth("", s.cfg.SMTPUser, s.cfg.SMTPPassword, s.cfg.SMTPHost)

	err := smtp.SendMail(addr, auth, from, []string{to}, []byte(msg))
	if err != nil {
		log.Printf("❌ Failed to send email to %s: %v", to, err)
		return err
	}

	log.Printf("📧 Email sent to %s: %s", to, subject)
	return nil
}
