# Погода в отчёте ГЭС — per-cascade контракт

Погода (`temperature`, `weather_condition`) хранится в отдельной таблице
`cascade_daily_data` на уровне каскада — одно значение на каскад в день.

## Схема БД

```sql
CREATE TABLE cascade_daily_data (
    id                BIGSERIAL PRIMARY KEY,
    organization_id   BIGINT NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    date              DATE NOT NULL,
    temperature       NUMERIC,
    weather_condition TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (organization_id, date)
);
```

`organization_id` ссылается на организацию-**каскад** (ту, у которой есть
запись в `cascade_config` с координатами).

## Как заполняется

Только фоновым планировщиком (`dayrotation.Service.fetchWeather`) —
запускается ежедневно в 04:00 Asia/Tashkent, снимает погоду через
OpenWeatherMap One Call 3.0 по координатам из `cascade_config` и апсертит
результат для каждого каскада.

Фронтенд **не** задаёт погоду. В `POST /api/v3/ges-report/daily-data` поля
`temperature`/`weather_condition` **отсутствуют** в DTO.

## В ответе `GET /api/v3/ges-report/?date=YYYY-MM-DD`

Погода живёт на `CascadeReport.weather`:

```json
{
  "cascades": [
    {
      "cascade_id": 10,
      "cascade_name": "Каскад X",
      "weather": {
        "temperature": 22.5,
        "weather_condition": "01d",
        "prev_year_temperature": 18.0
      },
      "summary": { ... },
      "stations": [ ... ]
    }
  ]
}
```

- `temperature` — °C, `float64 nullable`
- `weather_condition` — иконочный код OpenWeatherMap (`"10d"` = дождь днём,
  `"01n"` = ясно ночью), `string nullable`
- `prev_year_temperature` — температура за тот же день год назад, `float64
  nullable`

Станционные `StationReport.current` и `StationReport.previous_year` **не
содержат** погодных полей.
