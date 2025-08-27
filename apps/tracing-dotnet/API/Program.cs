using System.Diagnostics;
using Dapr.Client;
using OpenTelemetry.Resources;
using OpenTelemetry.Trace;

var builder = WebApplication.CreateBuilder(args);

// Add services to the container.

builder.Services.AddControllers();

// Add CORS services
builder.Services.AddCors(options =>
{
    options.AddPolicy("AllowLocalhost", policy =>
    {
        policy.SetIsOriginAllowed(origin =>
            Uri.TryCreate(origin, UriKind.Absolute, out var uri) &&
            (uri.Host == "localhost" || uri.Host == "127.0.0.1" || uri.Host.EndsWith(".localhost")))
              .AllowAnyMethod()
              .AllowAnyHeader()
              .AllowCredentials();
    });
});

// Configure multipart form options to handle large file uploads
builder.Services.Configure<Microsoft.AspNetCore.Http.Features.FormOptions>(options =>
{
    options.ValueLengthLimit = int.MaxValue;
    options.MultipartBodyLengthLimit = 1073741824; // 1 GB
    options.MultipartHeadersLengthLimit = int.MaxValue;
    options.ValueCountLimit = int.MaxValue;
    options.KeyLengthLimit = int.MaxValue;
});

// Configure request size limits
builder.Services.Configure<Microsoft.AspNetCore.Server.Kestrel.Core.KestrelServerOptions>(options =>
{
    options.Limits.MaxRequestBodySize = 1073741824; // 1 GB
});

// Uncomment this blob to fix telemetry issues
builder.Services.AddOpenTelemetry()
    .ConfigureResource(resource => resource.AddService("dsnext-api"))
    .WithTracing(tracing => tracing.AddAspNetCoreInstrumentation().AddZipkinExporter(options =>
    {
        options.Endpoint = new Uri("http://zipkin.default.svc.cluster.local:9411/api/v2/spans");
    }).SetSampler(new AlwaysOnSampler()));

builder.Services.AddDaprClient();

var app = builder.Build();

// Configure the HTTP request pipeline.

// app.UseHttpsRedirection();

// Enable CORS
app.UseCors("AllowLocalhost");

app.UseAuthorization();

app.UseRouting();
app.UseCloudEvents();

app.MapSubscribeHandler();
app.MapControllers();

//#if DEBUG
//Debugger.Launch();
//#endif

app.Run();
