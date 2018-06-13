package cc.dasa.forward;

import com.codesmyth.droidcook.api.Bundler;

@Bundler
public interface State {
  boolean listening();
  int port();
}
