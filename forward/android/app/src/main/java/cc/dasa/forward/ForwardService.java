package cc.dasa.forward;

import android.app.Service;
import android.content.Intent;
import android.os.IBinder;
import forward.Forward;

public class ForwardService extends Service {
  @Override
  public int onStartCommand(Intent intent, int flags, int startId) {
    handle(new StateBundler(intent));
    return START_STICKY;
  }

  @Override
  public IBinder onBind(Intent intent) {
    handle(new StateBundler(intent));
    return null;
  }

  void handle(State state) {
    try {
      if (state.listening()) {
        Forward.listenAndServe(":" + state.port());
        note("listening on :" + state.port());
      } else {
        Forward.close(":" + state.port());
        note("stopped listening on :" + state.port());
      }
    } catch (Exception e) {
      e.printStackTrace();
      note(e.getMessage());
    }
  }

  void note(String message) {
    new NoteBundler.Builder()
        .message(message)
        .broadcast(this);
  }
}