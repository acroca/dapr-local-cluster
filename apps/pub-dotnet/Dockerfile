# Build stage
FROM mcr.microsoft.com/dotnet/sdk:8.0-alpine AS build
WORKDIR /src

# Copy project file and restore dependencies
COPY checkout.csproj .
RUN dotnet restore checkout.csproj

# Copy source code and build
COPY . .
RUN dotnet build checkout.csproj -c Release -o /app/build

# Publish stage
FROM build AS publish
RUN dotnet publish checkout.csproj -c Release -o /app/publish /p:UseAppHost=false

# Runtime stage - Using aspnet runtime since the project uses Dapr.AspNetCore
FROM mcr.microsoft.com/dotnet/aspnet:8.0-alpine AS final
WORKDIR /app
COPY --from=publish /app/publish .
ENTRYPOINT ["dotnet", "checkout.dll"]
