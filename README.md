# autoMarket_AAA_workshop
Taller de curso de ciberseguridad, abarca una aplicacion para venta de vehiculos usados


marketplace/
├── cmd/
│   └── server/
│       └── main.go              # Punto de entrada. Inicializa Gin, la DB y arranca el servidor.
├── internal/
│   ├── models/                  # Estructuras de datos base (Entities)
│   │   ├── user.go              # Define ID, Name, Role (Admin, Vendedor, Cliente)
│   │   └── car.go               # Define propiedades del carro (Marca, Modelo, Precio, Estado)
│   │
│   ├── handlers/                # Controladores de Gin (Endpoints)
│   │   ├── user_handler.go      # Rutas: POST /users, GET /users/:id
│   │   └── car_handler.go       # Rutas: GET /cars, POST /cars
│   │
│   ├── middleware/              # Interceptores para Gin
│   │   ├── auth.go              # Verifica sesión o token JWT
│   │   └── rbac.go              # Control de Acceso Basado en Roles (Cliente, Vendedor, Admin)
│   │
│   ├── services/                # Lógica de negocio pura (Casos de uso)
│   │   ├── user_service.go      # Lógica de registro, validación de permisos
│   │   └── car_service.go       # Lógica de publicación, aprobación de carros
│   │
│   └── repositories/            # Capa custom de acceso a datos (Queries SQL puras)
│       ├── user_repo.go         # Ejecuta INSERT/SELECT sobre la tabla users
│       └── car_repo.go          # Ejecuta queries de los vehículos
│
├── pkg/                         # Paquetes genéricos reutilizables (no atados al dominio)
│   ├── database/
│   │   └── connection.go        # Inicialización de la conexión SQL
│   └── response/
│       └── http_response.go     # Estandarización de respuestas JSON de Gin
│
├── web/                         # Archivos del Frontend (Si usas Go para servir la vista)
│   ├── static/                  # CSS, JS, Imágenes
│   └── templates/               # Archivos .html (si usas html/template con Gin)
│
├── config/
│   └── config.go                # Carga de variables de entorno (.env)
│
├── go.mod
└── go.sum