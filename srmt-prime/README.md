# ac-integration-gitops

GitOps репозиторий для ac-integration.

## Структура

```
├── base/                    # Базовые манифесты
│   ├── deployment.yaml
│   ├── service.yaml
│   └── kustomization.yaml
├── overlays/
│   ├── dev/                 # Dev окружение
│   │   └── kustomization.yaml
│   └── prod/                # Prod окружение
│       └── kustomization.yaml
└── apps/                    # ArgoCD Applications
    ├── ac-integration-dev.yaml
    └── ac-integration-prod.yaml
```

## Деплой ArgoCD Applications

```bash
kubectl apply -f apps/
```

## Доступ к сервису

```
# Из namespace dev/prod
http://ac-integration/

# Из другого namespace
http://ac-integration.dev/
http://ac-integration.prod/
```
