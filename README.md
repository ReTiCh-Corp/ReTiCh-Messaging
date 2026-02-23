# ReTiCh Messaging Service

Service de messagerie temps réel pour la plateforme ReTiCh. Gère les conversations, messages et WebSockets.

## Fonctionnalités

- Messages directs (1-to-1)
- Conversations de groupe
- Channels publics
- WebSocket temps réel
- Indicateurs de frappe
- Accusés de lecture
- Réactions aux messages
- Pièces jointes (images, fichiers, audio, vidéo)
- Messages épinglés

## Prérequis

- Go 1.22+
- PostgreSQL 16+
- Redis
- NATS
- Docker (optionnel)

## Démarrage rapide

### Avec Docker (recommandé)

```bash
# Depuis le repo ReTiCh-Infrastucture
make up
make migrate-messaging
```

### Sans Docker

```bash
# Installer les dépendances
go mod download

# Configurer les services
export DATABASE_URL="postgres://retich:retich_secret@localhost:5433/retich_messaging?sslmode=disable"
export NATS_URL="nats://localhost:4222"
export REDIS_URL="localhost:6379"

# Lancer les migrations
migrate -path migrations -database "$DATABASE_URL" up

# Lancer le serveur
go run cmd/server/main.go
```

### Développement avec hot-reload

```bash
# Installer Air
go install github.com/air-verse/air@latest

# Lancer avec hot-reload
air -c .air.toml
```

## Configuration

Variables d'environnement:

| Variable | Description | Défaut |
|----------|-------------|--------|
| `PORT` | Port du serveur | `8082` |
| `DATABASE_URL` | URL PostgreSQL | - |
| `NATS_URL` | URL NATS | `nats://nats:4222` |
| `REDIS_URL` | URL Redis | `redis:6379` |
| `LOG_LEVEL` | Niveau de log | `info` |

## Endpoints

### REST API

| Méthode | Endpoint | Description |
|---------|----------|-------------|
| GET | `/health` | Health check |
| GET | `/ready` | Readiness check |
| GET | `/conversations` | Liste des conversations |
| POST | `/conversations` | Créer une conversation |
| GET | `/conversations/:id` | Détails d'une conversation |
| GET | `/conversations/:id/messages` | Messages d'une conversation |
| POST | `/conversations/:id/messages` | Envoyer un message |
| PUT | `/messages/:id` | Modifier un message |
| DELETE | `/messages/:id` | Supprimer un message |
| POST | `/messages/:id/reactions` | Ajouter une réaction |
| DELETE | `/messages/:id/reactions/:emoji` | Retirer une réaction |
| POST | `/conversations/:id/read` | Marquer comme lu |

### WebSocket

```
ws://localhost:8082/ws
```

Événements:
- `message.new` - Nouveau message
- `message.edit` - Message modifié
- `message.delete` - Message supprimé
- `typing.start` - Utilisateur tape
- `typing.stop` - Utilisateur arrête de taper
- `presence.update` - Changement de statut

## Base de données

### Schéma (optimisé pour 500K msg/sec)

```
conversations
├── id (UUID, PK)
├── type (direct/group/channel)
├── name
├── description
├── avatar_url
├── creator_id
├── is_archived
├── last_message_at
└── timestamps

conversation_participants
├── conversation_id (FK)
├── user_id
├── role (owner/admin/member)
├── nickname
├── is_muted
├── muted_until
├── last_read_at
├── last_read_message_id
├── joined_at
└── left_at

messages (PARTITIONED BY RANGE created_at)
├── id (UUID)
├── conversation_id
├── sender_id
├── type (text/image/file/audio/video/system)
├── content
├── metadata (JSONB)
├── reply_to_id
├── is_edited
├── is_deleted
└── created_at

attachments
├── message_id (FK)
├── file_name
├── file_type
├── file_size
├── file_url
├── thumbnail_url
└── dimensions/duration

message_reactions
├── message_id (FK)
├── user_id
└── emoji

read_receipts
├── conversation_id (FK)
├── user_id
├── last_read_message_id
└── last_read_at

pinned_messages
├── conversation_id (FK)
├── message_id (FK)
├── pinned_by
└── pinned_at
```

### Partitionnement

La table `messages` est partitionnée par mois pour des performances optimales:
- `messages_y2026m02`
- `messages_y2026m03`
- `messages_y2026m04`
- `messages_default` (fallback)

### Migrations

```bash
# Appliquer les migrations
migrate -path migrations -database "$DATABASE_URL" up

# Rollback
migrate -path migrations -database "$DATABASE_URL" down 1
```

## Architecture temps réel

```
┌─────────┐     ┌─────────┐     ┌─────────┐
│ Client  │────▶│   API   │────▶│   DB    │
└─────────┘     └────┬────┘     └─────────┘
     ▲               │
     │          ┌────▼────┐
     │          │  NATS   │
     │          └────┬────┘
     │               │
     └───────────────┘
         WebSocket
```

- Les messages sont persistés en PostgreSQL
- NATS distribue les événements temps réel
- Redis gère les indicateurs de frappe (TTL court)
- WebSocket pousse les événements aux clients

## Structure du projet

```
ReTiCh-Messaging/
├── cmd/
│   └── server/
│       └── main.go         # Point d'entrée
├── internal/               # Code interne
├── migrations/
│   ├── 000001_init_schema.up.sql
│   └── 000001_init_schema.down.sql
├── Dockerfile              # Image production
├── Dockerfile.dev          # Image développement
├── .air.toml               # Config hot-reload
├── go.mod
└── go.sum
```

## Tests

```bash
# Lancer les tests
go test ./...

# Avec couverture
go test -cover ./...
```

## Performance

Optimisations pour 100K connexions / 500K msg/sec:
- Partitionnement mensuel des messages
- Index optimisés sur les requêtes fréquentes
- Connection pooling PostgreSQL
- NATS pour la distribution d'événements
- Redis pour le cache et les données éphémères

## Licence

MIT
