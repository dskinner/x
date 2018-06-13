package cc.dasa.forward;

import android.content.Intent;
import android.databinding.DataBindingUtil;
import android.os.Bundle;
import android.support.design.widget.Snackbar;
import android.support.v7.app.AppCompatActivity;
import android.view.View;
import cc.dasa.forward.databinding.ForwardActivityBinding;
import com.codesmyth.droidcook.PackedReceiver;
import com.codesmyth.droidcook.ReceiverFactory;
import com.codesmyth.droidcook.api.Action;

public class ForwardActivity extends AppCompatActivity {

  PackedReceiver receiver = ReceiverFactory.makeFor(this);
  StateBundler state = new StateBundler.Builder().port(9090).bundle();
  ForwardActivityBinding binding;

  @Override
  protected void onCreate(Bundle savedInstanceState) {
    super.onCreate(savedInstanceState);
    state.edit().readFrom(savedInstanceState);
    binding = DataBindingUtil.setContentView(this, R.layout.forward_activity);
    binding.setState(state);
    receiver.register(this);
  }

  @Override
  protected void onDestroy() {
    super.onDestroy();
    receiver.unregister(this);
  }

  @Override
  protected void onSaveInstanceState(Bundle outState) {
    state.writeTo(outState);
    super.onSaveInstanceState(outState);
  }

  @Override
  protected void onRestoreInstanceState(Bundle savedInstanceState) {
    super.onRestoreInstanceState(savedInstanceState);
    state.edit().readFrom(savedInstanceState);
    binding.setState(state);
  }

  @Action
  public void note(Note note) {
    Snackbar.make(binding.content, note.message(), Snackbar.LENGTH_LONG).show();
  }

  public void toggleListening(View v) {
    state.edit().listening(!state.listening());
    binding.setState(state);

    Intent intent = new Intent(this, ForwardService.class);
    state.writeTo(intent);
    startService(intent);
  }
}
