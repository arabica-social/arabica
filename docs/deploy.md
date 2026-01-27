# Arabica Deployment Guide

Quick guide to deploy Arabica to a VPS with Docker and automatic HTTPS.

## Prerequisites

- VPS with Docker and Docker Compose installed
- Domain name pointing to your VPS IP address (A record)
- Ports 80 and 443 open in firewall

## Quick Start (Production)

1. **Clone the repository:**
   ```bash
   git clone <repository-url>
   cd arabica
   ```

2. **Configure your domain:**
   ```bash
   cp .env.example .env
   nano .env
   ```
   
   Update `.env` with your domain:
   ```env
   DOMAIN=arabica.yourdomain.com
   ACME_EMAIL=your-email@example.com
   ```

3. **Deploy:**
   ```bash
   docker compose up -d
   ```

That's it! Caddy will automatically:
- Obtain SSL certificates from Let's Encrypt
- Renew certificates before expiry
- Redirect HTTP to HTTPS
- Proxy requests to Arabica

4. **Check logs:**
   ```bash
   docker compose logs -f
   ```

5. **Visit your site:**
   ```
   https://arabica.yourdomain.com
   ```

## Local Development

To run locally without a domain:

```bash
docker compose up
```

Then visit `http://localhost` (Caddy will serve on port 80).

## Updating

```bash
git pull
docker compose down
docker compose build
docker compose up -d
```

## Troubleshooting

### Certificate Issues

If Let's Encrypt can't issue a certificate:
- Ensure your domain DNS is pointing to your VPS
- Check ports 80 and 443 are accessible
- Check logs: `docker compose logs caddy`

### View Arabica logs

```bash
docker compose logs -f arabica
```

### Reset everything

```bash
docker compose down -v  # Warning: deletes all data
docker compose up -d
```

## Production Checklist

- [ ] Domain DNS pointing to VPS
- [ ] Ports 80 and 443 open in firewall
- [ ] `.env` file configured with your domain
- [ ] Valid email set for Let's Encrypt notifications
- [ ] Regular backups of `arabica-data` volume

## Backup

To backup user data:

```bash
docker compose exec arabica cp /data/arabica.db /data/arabica-backup.db
docker cp $(docker compose ps -q arabica):/data/arabica-backup.db ./backup-$(date +%Y%m%d).db
```

## Advanced Configuration

### Custom Caddyfile

Edit `Caddyfile` directly for advanced options like:
- Rate limiting
- Custom headers
- IP whitelisting
- Multiple domains

### Environment Variables

All available environment variables in `.env`:

| Variable            | Default                              | Description                     |
| ------------------- | ------------------------------------ | ------------------------------- |
| `DOMAIN`            | localhost                            | Your domain name                |
| `ACME_EMAIL`        | (empty)                              | Email for Let's Encrypt         |
| `LOG_LEVEL`         | info                                 | debug/info/warn/error           |
| `LOG_FORMAT`        | json                                 | console/json                    |
| `SERVER_PUBLIC_URL` | https://${DOMAIN}                    | Override public URL (enables secure cookies when HTTPS) |

## Support

For issues, check the logs first:
```bash
docker compose logs
```
