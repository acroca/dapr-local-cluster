using System.Diagnostics;
using Dapr.Messaging.PublishSubscribe.Extensions;

var builder = WebApplication.CreateBuilder(args);

// Add services to the container.

builder.Services.AddControllers();

builder.Services.AddDaprPubSubClient();

builder.Services.AddDaprClient();

var app = builder.Build();

// Configure the HTTP request pipeline.

app.UseHttpsRedirection();

app.UseAuthorization();

app.UseRouting();
app.UseCloudEvents();

app.MapSubscribeHandler();
app.MapControllers();

//#if DEBUG
//Debugger.Launch();
//#endif

app.Run();
