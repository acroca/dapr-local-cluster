"""Playground for the gRPC actor stream (alpha) feature of the Python SDK.

The actor is hosted with ActorGrpcHost: instead of exposing HTTP actor callback
endpoints, the app dials daprd's gRPC port and receives every actor callback
(invocations, reminders, timers, deactivations) over a single app-initiated
SubscribeActorEventsAlpha1 stream.

PlaygroundActor implements every available callback and prints when each fires:

- _on_activate / _on_deactivate                       actor lifecycle
- _on_pre_actor_method / _on_post_actor_method        around every invocation
- receive_reminder                                    reminder fires (Remindable)
- timer_callback                                      timer fires

Invoke methods through the sidecar HTTP API (port-forwarded by Tilt to 3511):

    curl -X POST localhost:3511/v1.0/actors/PlaygroundActor/demo/method/Hello -d '"world"'
"""

import asyncio
import os
from datetime import timedelta
from typing import Optional

import grpc.aio  # type: ignore

from dapr.actor import Actor, ActorGrpcHost, ActorInterface, Remindable, actormethod
from dapr.actor.runtime._method_context import ActorMethodContext
from dapr.actor.runtime.config import ActorRuntimeConfig
from dapr.actor.runtime.runtime import ActorRuntime

APP_PORT = int(os.getenv('APP_PORT', '6011'))


class PlaygroundActorInterface(ActorInterface):
    @actormethod(name='Hello')
    async def hello(self, name: object) -> str: ...

    @actormethod(name='GetData')
    async def get_data(self) -> object: ...

    @actormethod(name='SetData')
    async def set_data(self, data: object) -> None: ...

    @actormethod(name='StartReminder')
    async def start_reminder(self) -> None: ...

    @actormethod(name='StopReminder')
    async def stop_reminder(self) -> None: ...

    @actormethod(name='StartTimer')
    async def start_timer(self) -> None: ...

    @actormethod(name='StopTimer')
    async def stop_timer(self) -> None: ...


class PlaygroundActor(Actor, PlaygroundActorInterface, Remindable):
    """Exercises every actor callback over the gRPC stream, printing each one."""

    def _say(self, message: str) -> None:
        print(f'[{type(self).__name__}/{self.id}] {message}', flush=True)

    # --- lifecycle callbacks ---

    async def _on_activate(self) -> None:
        self._say('_on_activate')

    async def _on_deactivate(self) -> None:
        self._say('_on_deactivate')

    # --- invocation hooks (fire around every method, reminder and timer) ---

    async def _on_pre_actor_method(self, method_context: ActorMethodContext) -> None:
        self._say(f'_on_pre_actor_method: {method_context.method_name}')

    async def _on_post_actor_method(self, method_context: ActorMethodContext) -> None:
        self._say(f'_on_post_actor_method: {method_context.method_name}')

    # --- actor methods ---

    async def hello(self, name: object) -> str:
        self._say(f'Hello: {name!r}')
        return f'Hello {name} from {self.id}!'

    async def get_data(self) -> object:
        has_value, data = await self._state_manager.try_get_state('data')
        self._say(f'GetData: has_value={has_value} data={data!r}')
        return data

    async def set_data(self, data: object) -> None:
        self._say(f'SetData: {data!r}')
        await self._state_manager.set_state('data', data)
        await self._state_manager.save_state()

    # --- reminders ---

    async def start_reminder(self) -> None:
        self._say('StartReminder: registering "playground_reminder" (due 5s, period 10s)')
        await self.register_reminder(
            'playground_reminder',
            b'reminder-state',
            timedelta(seconds=5),
            timedelta(seconds=10),
        )

    async def stop_reminder(self) -> None:
        self._say('StopReminder: unregistering "playground_reminder"')
        await self.unregister_reminder('playground_reminder')

    async def receive_reminder(
        self,
        name: str,
        state: bytes,
        due_time: timedelta,
        period: timedelta,
        ttl: Optional[timedelta] = None,
    ) -> None:
        self._say(
            f'receive_reminder: name={name} state={state!r} '
            f'due_time={due_time} period={period} ttl={ttl}'
        )

    # --- timers ---

    async def start_timer(self) -> None:
        self._say('StartTimer: registering "playground_timer" (due 5s, period 10s)')
        await self.register_timer(
            'playground_timer',
            self.timer_callback,
            'timer-state',
            timedelta(seconds=5),
            timedelta(seconds=10),
        )

    async def stop_timer(self) -> None:
        self._say('StopTimer: unregistering "playground_timer"')
        await self.unregister_timer('playground_timer')

    async def timer_callback(self, state: object) -> None:
        self._say(f'timer_callback: state={state!r}')


async def main() -> None:
    # Short idle timeout so _on_deactivate shows up quickly while playing around.
    ActorRuntime.set_actor_config(
        ActorRuntimeConfig(
            actor_idle_timeout=timedelta(seconds=60),
            actor_scan_interval=timedelta(seconds=10),
        )
    )

    # daprd requires a gRPC app channel (--app-protocol grpc --app-port) before it
    # accepts the actor event stream, so listen on the app port with an empty gRPC
    # server. No actor traffic arrives here: every callback comes over the stream.
    app_server = grpc.aio.server()
    app_server.add_insecure_port(f'[::]:{APP_PORT}')
    await app_server.start()

    host = ActorGrpcHost()
    await host.register_actor(PlaygroundActor)
    print(
        f'{PlaygroundActor.__name__} hosted over the Dapr gRPC actor stream '
        f'(app port {APP_PORT})',
        flush=True,
    )
    try:
        await host.run_forever()
    finally:
        await app_server.stop(grace=None)


if __name__ == '__main__':
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        pass
