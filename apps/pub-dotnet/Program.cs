using System;
using Dapr.Client;
using System.Diagnostics;
using System.Text.Json.Serialization;
using System.Threading.Tasks;


using var client = new DaprClientBuilder().Build();

// ActivitySource source = new ActivitySource("pub-dotnet-as", "1.0");

// for (int i = 1; i > 0; i++) {
//     var order = new Order(i);

//     using (var activity = source.StartActivity("event-publishing"))
//     {
//       await client.PublishEventAsync("pubsub", "numbers", order);
//       Console.WriteLine("Published data: " + order);
//       await Task.Delay(TimeSpan.FromSeconds(1));
//       await client.PublishEventAsync("pubsub", "numbers", order);
//       Console.WriteLine("Published data: " + order);
//     }

//     await Task.Delay(TimeSpan.FromSeconds(1));
// }



for (int i = 1; i > 0; i++) {
    var order = new Order(i);
    using var activity = new Activity("event-publishing");
    activity.SetIdFormat(ActivityIdFormat.W3C);
    activity.Start();

    await client.PublishEventAsync("pubsub", "numbers", order);
    Console.WriteLine("Published data: " + order);

    await Task.Delay(TimeSpan.FromSeconds(1));
}

public record Order([property: JsonPropertyName("orderId")] int OrderId);
